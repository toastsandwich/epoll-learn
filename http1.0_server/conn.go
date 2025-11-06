package server

import (
	"time"

	"github.com/toastsandwich/epoll-learn/http1.0_server/pkg/pool"
	"golang.org/x/sys/unix"
)

func OnReadable(c *Conn) (int, error) {
	totalBytes := 0
	for {
		stop, n, err := c.Recv()
		if err != nil {
			return totalBytes, err
		}
		if stop {
			break
		}
		totalBytes += n
	}
	return totalBytes, nil
}

func OnWriteable(c *Conn) (int, error) {
	totalBytes := 0
	for {
		stop, n, err := c.Send()
		if err != nil {
			return totalBytes, err
		}
		if stop {
			break
		}
		totalBytes += n
	}
	return totalBytes, nil
}

type Conn struct {
	// One request at a time
	ReadBuffer  []byte
	WriteBuffer []byte

	fd      int // conn fd
	epollfd int // epoll loop

	aliveAt time.Time
}

func NewConn(fd int) *Conn {
	c := &Conn{}

	c.ReadBuffer = pool.GetBuffer()
	c.WriteBuffer = pool.GetBuffer()

	c.aliveAt = time.Now()
	return c
}

// Send and Recv both return an extra parameter to notify loop to continue
func (c *Conn) Recv() (bool, int, error) {
	n, err := unix.Read(c.fd, c.ReadBuffer)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return true, 0, nil
		}
		return true, 0, err
	}
	return false, n, nil
}

func (c *Conn) SendJob() {
	job := Job{}
}

func (c *Conn) Send() (bool, int, error) {
	if len(c.WriteBuffer) == 0 {
		return true, 0, nil
	}

	n, err := unix.Write(c.fd, c.WriteBuffer)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return true, 0, nil
		}
		return true, 0, err
	}
	if n < len(c.WriteBuffer) {
		c.WriteBuffer = c.WriteBuffer[n:]
		return false, n, nil
	}
	return false, n, err
}

func (c *Conn) Close() {
	pool.PutBuffer(c.ReadBuffer)
	pool.PutBuffer(c.WriteBuffer)
	unix.Close(c.fd)
}
