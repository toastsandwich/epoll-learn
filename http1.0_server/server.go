package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/toastsandwich/epoll-learn/http1.0_server/pkg/pool"
	"golang.org/x/sys/unix"
)

const (
	EVENT_IN_OUT_ET_ERR = unix.EPOLLIN | unix.EPOLLOUT | unix.EPOLLET | unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLRDHUP
	EVENT_IN_ET_ERR     = unix.EPOLLIN | unix.EPOLLET | unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLRDHUP
)

type HTTPServerOpts struct {
	Addr string
	Port int

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type HTTPServer struct {
	sockAddr unix.SockaddrInet4

	Fd      int // server fd
	epollFd int

	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	ActiveConnMap map[int]*Conn

	mu sync.RWMutex
}

func NewHTTPServer(opts *HTTPServerOpts) (*HTTPServer, error) {
	server := &HTTPServer{}

	fmt.Sscanf(opts.Addr, "%d.%d.%d.%d", server.sockAddr.Addr[0], server.sockAddr.Addr[1], server.sockAddr.Addr[2], server.sockAddr.Addr[3])
	server.sockAddr.Port = opts.Port

	server.ActiveConnMap = make(map[int]*Conn)

	server.ReadTimeout = opts.ReadTimeout
	server.WriteTimeout = opts.WriteTimeout

	if err := server.initSocket(); err != nil {
		return nil, err
	}
	if err := server.setUpEPolling(); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *HTTPServer) initSocket() error {

	sockfd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		return err
	}

	if err := unix.SetsockoptInt(sockfd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return err
	}
	if err := unix.SetsockoptInt(sockfd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return err
	}

	if err := unix.SetNonblock(sockfd, true); err != nil {
		return err
	}

	if err := unix.Bind(sockfd, &s.sockAddr); err != nil {
		return err
	}

	s.Fd = sockfd
	return nil
}

func (s *HTTPServer) setUpEPolling() error {
	epollFd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return err
	}

	// first event always given to http server.
	event := &unix.EpollEvent{
		Fd:     int32(s.Fd),
		Events: EVENT_IN_ET_ERR,
	}

	if err := unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, s.Fd, event); err != nil {
		return err
	}
	s.epollFd = epollFd
	return nil
}

func (s *HTTPServer) accept() {
	for {
		cfd, _, err := unix.Accept(s.Fd)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}
			fmt.Println("error accepting new connection:", err)
			continue
		}
		fmt.Println("new connection!!")

		if err := unix.SetNonblock(cfd, true); err != nil {
			fmt.Println(err)
			continue
		}

		// create conn and add it to conn map
		conn := NewConn(cfd)
		s.mu.Lock()
		s.ActiveConnMap[cfd] = conn
		s.mu.Unlock()

		// hand off cfd as intrested in epoll instance
		clientEvent := &unix.EpollEvent{
			Fd:     int32(cfd),
			Events: EVENT_IN_ET_ERR,
		}

		if err := unix.EpollCtl(s.epollFd, unix.EPOLL_CTL_ADD, cfd, clientEvent); err != nil {
			fmt.Println("error setting controls for new client connection:", err)
		}
	}
}

func (s *HTTPServer) ListenAndServe() error {

	if err := unix.Listen(s.Fd, 2048); err != nil {
		return err
	}

	fmt.Println("server online")
	epollEvents := make([]unix.EpollEvent, 1000)

	for {
		n, err := unix.EpollWait(s.epollFd, epollEvents, -1)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return fmt.Errorf("error during epoll wait:%v", err)
		}
		for i := range n {
			e := epollEvents[i]

			switch {
			case e.Fd == int32(s.Fd):
				s.accept()

			case e.Events&(unix.EPOLLERR|unix.EPOLLHUP|unix.EPOLLRDHUP) != 0:
				s.mu.RLock()
				if conn, ok := s.ActiveConnMap[int(e.Fd)]; ok {
					conn.Close()
					delete(s.ActiveConnMap, int(e.Fd))
				}
				s.mu.RUnlock()

				errno, err := unix.GetsockoptInt(int(e.Fd), unix.SOL_SOCKET, unix.SO_ERROR)
				if err != nil {
					return err
				}
				err = unix.Errno(errno)

				fmt.Printf("errored for fd=%d err=%v\n", e.Fd, err)

			case e.Events&(unix.EPOLLIN) != 0:
				for {
					buf := pool.GetBuffer()
					n, err := unix.Read(int(e.Fd), buf)
					if err != nil {
						if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
							break
						}
					}
					if n == -1 {
						break
					}
					fmt.Println("read", n, "bytes worth data")
					// send this buffer to worker
					// work should parse the data, clean it and send a response back
					// worker will also change the event for OUT
					//

					data := make([]byte, n)
					copy(data, buf[:n])
					pool.PutBuffer(buf)
					// remember now EPOLLOUT is one of interested events
					if err := unix.EpollCtl(s.epollFd, unix.EPOLL_CTL_MOD, int(e.Fd), &unix.EpollEvent{
						Events: EVENT_IN_OUT_ET_ERR,
						Fd:     e.Fd,
					}); err != nil {
						fmt.Println("error modifying intrest list:", err)
					}
					s.jm.toInChan(int(e.Fd), data)
				}
			case e.Events&unix.EPOLLOUT != 0:
				for {
					data := MasterRecords.Submit(int(e.Fd))
					if data == nil {
						continue
					}
					written := 0
					for written < len(data) {
						n, err := unix.Write(int(e.Fd), data)
						if err != nil {
							if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
								MasterRecords.Add(int(e.Fd), data[n:])
								break
							}
							if err == unix.ECONNRESET || err == unix.EPIPE {
								s.CloseClient(int(e.Fd))
							}
							fmt.Println("error writing data:", err)
							continue
						}
						written += n
					}
					// after submit, update event list
					if err := unix.EpollCtl(s.epollFd, unix.EPOLL_CTL_MOD, int(e.Fd), &unix.EpollEvent{
						Fd:     e.Fd,
						Events: EVENT_IN_ET_ERR,
					}); err != nil {
						fmt.Println("err modifying event:", err)
					}

				}
			}
		}
	}
}

func (s *HTTPServer) Close() error {
	if err := unix.EpollCtl(s.epollFd, unix.EPOLL_CTL_DEL, s.Fd, nil); err != nil {
		return err
	}
	return unix.Close(s.Fd)
}

func (s *HTTPServer) CloseClient(fd int) error {
	if err := unix.EpollCtl(s.epollFd, unix.EPOLL_CTL_DEL, fd, nil); err != nil {
		return err
	}
	return unix.Close(fd)
}
