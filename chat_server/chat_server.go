package main

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

const MAXACTIVECONNS = 100_000

func SockIPv4ToString(s unix.Sockaddr) string {
	switch a := s.(type) {
	case *unix.SockaddrInet4:
		return fmt.Sprintf("%d.%d.%d.%d:%d", a.Addr[0], a.Addr[1], a.Addr[2], a.Addr[3], a.Port)
	default:
		return "unknown socket type"
	}
}

type ChatServer struct {
	SocketAddrInet4 unix.SockaddrInet4

	Fd      int // fd for server
	epollFd int // fd for epoll instance

	ActiveUserMap map[int]string

	bp *BufferPool

	mu sync.Mutex
}

func NewChatServer(a, b, c, d byte, port int) *ChatServer {
	ch := &ChatServer{}

	addr := [4]byte{a, b, c, d}
	copy(ch.SocketAddrInet4.Addr[:], addr[:])
	ch.SocketAddrInet4.Port = port

	ch.ActiveUserMap = make(map[int]string)

	ch.bp = NewBufferPool(true)
	ifErrExit(ch.bindAndListen(), "error binding and listening")
	ifErrExit(ch.setupEpollAndEvents(), "error setting up epoll and its events")
	return ch
}

func (c *ChatServer) bindAndListen() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	c.Fd = fd

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return err
	}

	if err := unix.SetNonblock(fd, true); err != nil {
		return err
	}

	fmt.Println("chat server started to listen on", SockIPv4ToString(&c.SocketAddrInet4))

	if err := unix.Bind(fd, &c.SocketAddrInet4); err != nil {
		return err
	}

	return unix.Listen(fd, 2048) // max pending connections can be 2048
}

func (c *ChatServer) setupEpollAndEvents() error {
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return err
	}

	c.epollFd = epfd

	serverAcceptEvent := &unix.EpollEvent{
		Fd:     int32(c.Fd),
		Events: unix.EPOLLIN | unix.EPOLLERR | unix.EPOLLRDHUP | unix.EPOLLHUP,
	}
	return unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, c.Fd, serverAcceptEvent)
}

func (c *ChatServer) accept() error {
	cfd, csockaddr, err := unix.Accept(c.Fd)
	if err != nil {
		return err
	}
	sockString := SockIPv4ToString(csockaddr)
	fmt.Println("new connection from: ", sockString)

	c.mu.Lock()
	c.ActiveUserMap[cfd] = sockString
	c.mu.Unlock()

	return unix.EpollCtl(c.epollFd, unix.EPOLL_CTL_ADD, cfd, &unix.EpollEvent{
		Fd:     int32(cfd),
		Events: unix.EPOLLERR | unix.EPOLLIN | unix.EPOLLRDHUP | unix.EPOLLHUP,
	})
}

// Serve, accepts incoming connection, read messages from them and broadcasts them.
func (c *ChatServer) Serve() {
	// serve waits for events for happens and adjusts
	for {
		events := make([]unix.EpollEvent, 1000)
		n, err := unix.EpollWait(c.epollFd, events, -1) // wait forever
		// ifErrExit(err, "error waiting for events to happen")
		if err != nil {
			if err == unix.EINTR {
				fmt.Println("error waiting for events:", err)
				continue
			}
		}
		for i := range n {
			event := events[i]
			evtFd, evts := event.Fd, event.Events
			switch {
			case evtFd == int32(c.Fd):
				if err := c.accept(); err != nil {
					fmt.Println("error accepting connection:", err)
				}

			case evts&unix.EPOLLERR != 0:
				errNo, err := unix.GetsockoptInt(int(evtFd), unix.SOL_SOCKET, unix.SO_ERROR)
				if err != nil {
					fmt.Println("error getting socket error number:", err)
				} else {
					fmt.Println("error from epoll:", unix.Errno(errNo))
				}
				c.CloseClient(int(evtFd))
			case evts&unix.EPOLLHUP != 0:
			case evts&unix.EPOLLRDHUP != 0: // this means that client has closed the connection.
				c.CloseClient(int(evtFd))

			case evts&unix.EPOLLIN != 0:
				buf := c.bp.GetBuffer()
				n, err := unix.Read(int(evtFd), buf)
				if err != nil {
					fmt.Println("error reading from cient:", err)
					continue
				}
				c.broadcast(int(evtFd), buf[:n])
				c.bp.PutBuffer(buf)
				// default:
				// fmt.Println("no matching casee")
			}
		}
	}
}

func (c *ChatServer) broadcast(from int, b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.ActiveUserMap {
		if k == from {
			continue
		}
		msg := fmt.Sprintf("%s: %s", c.ActiveUserMap[from], string(b))
		_, err := unix.Write(k, []byte(msg))
		if err != nil {
			fmt.Println("error writing message to client:", err)
			continue
		}
	}
}

func (c *ChatServer) Close() {
	unix.Close(c.epollFd)
	unix.Close(c.Fd)
}

// close client handles the remove from epoll intrest list as well.
func (c *ChatServer) CloseClient(fd int) {
	unix.EpollCtl(c.epollFd, unix.EPOLL_CTL_DEL, fd, nil)
	unix.Close(fd)

	c.mu.Lock()
	delete(c.ActiveUserMap, fd)
	c.mu.Unlock()
}

func ifErrExit(err error, msg string) {
	if err != nil {
		fmt.Println(msg+":", err)
		os.Exit(1)
	}
}
