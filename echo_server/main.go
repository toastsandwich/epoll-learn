package main

import (
	"fmt"
	"io"
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
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	ifError(err)
	defer unix.Close(fd)

	ifError(unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1))

	addr := unix.SockaddrInet4{Port: 9000}
	copy(addr.Addr[:], []byte{0, 0, 0, 0})

	ifError(unix.Bind(fd, &addr))

	unix.Listen(fd, 10)

	handleConn := func(fd int) {
		for {
			buf := bufferPool.get()

			n, err := unix.Read(fd, buf)
			if err != nil {
				if err == io.EOF {
					unix.Write(fd, []byte("bye"))
					return
				}
				fmt.Println(err)
			}
			unix.Write(fd, buf[:n])
			bufferPool.put(buf)
		}
	}

	for {
		cfd, sockaddr, err := unix.Accept(fd)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("recieved connection from:", sockaddr.(*unix.SockaddrInet4).Addr)
		go handleConn(cfd)
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
