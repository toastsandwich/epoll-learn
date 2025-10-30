package main

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

const BUFFEERSIZE = 4096

type BufferPool struct {
	sync.Pool
	Secure bool
}

func CreateBufferPool(secure bool) *BufferPool {
	return &BufferPool{
		sync.Pool{
			New: func() any {
				fmt.Println("buffer pool just create a new buffer")
				return make([]byte, BUFFEERSIZE)
			},
		},
		secure,
	}
}

func (b *BufferPool) get() []byte {
	return b.Get().([]byte)
}

func (b *BufferPool) put(p []byte) {
	if b.Secure {
		for i := range p {
			p[i] = 0
		}
	}
	b.Put(p)
}

var bufferPool = CreateBufferPool(false)

func socket_bind_listen() {
	srvfd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	ifError(err)
	defer unix.Close(srvfd)
	unix.SetNonblock(srvfd, true) // set server fd to non block so it doesnot block forever while reading

	ifError(unix.SetsockoptInt(srvfd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1))

	addr := unix.SockaddrInet4{Port: 9000}
	copy(addr.Addr[:], []byte{0, 0, 0, 0})

	ifError(unix.Bind(srvfd, &addr))
	ifError(unix.Listen(srvfd, 10))
	fmt.Println("echo server is now listening at 0.0.0.0:9000")
	// this is the epoll instance fd
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	ifError(err)

	// now register events to this epfd
	eventTypes := unix.EPOLLIN | unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLRDHUP
	ifError(unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, srvfd, &unix.EpollEvent{
		Events: uint32(eventTypes),
		Fd:     int32(srvfd),
	}))

	events := make([]unix.EpollEvent, 100) // monitor at max 100 events
	for {
		n, err := unix.EpollWait(epfd, events, -1) // block forever
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			ifError(err)
		}
		for i := range n {
			evt := events[i]
			efd := int(evt.Fd) // this is event fd
			if evt.Events&(unix.EPOLLERR|unix.EPOLLHUP|unix.EPOLLRDHUP) != 0 {
				unix.Close(efd)
				unix.EpollCtl(epfd, unix.EPOLL_CTL_DEL, efd, nil)
				continue
			}
			// if the events is of server fd, then we have an incoming connection.
			if efd == srvfd {
				clntfd, csockaddr, err := unix.Accept(efd)
				if err != nil {
					fmt.Println(err)
					continue
				}

				// now add the client fd to event poll
				ifError(unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, clntfd, &unix.EpollEvent{
					Events: uint32(eventTypes),
					Fd:     int32(clntfd),
				}))

				fmt.Println("new connection from", csockaddr.(*unix.SockaddrInet4).Addr)
			} else { // we are getting data from clients, get a buffer from poll and read the data and echo it
				buf := bufferPool.get()
				n, err := unix.Read(efd, buf)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if n == 0 { // this means that client is done with server
					unix.Close(efd)
					ifError(unix.EpollCtl(epfd, unix.EPOLL_CTL_DEL, efd, nil))
					bufferPool.put(buf)
					continue
				}
				_, err = unix.Write(efd, buf[:n])
				if err != nil {
					fmt.Println(err)
					bufferPool.put(buf)
					continue
				}
				bufferPool.put(buf)
			}
		}
	}
}

func ifError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	socket_bind_listen()
}
