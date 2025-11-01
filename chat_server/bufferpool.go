package main

import (
	"sync"
)

const MAXBUFFERSIZE = 4096

type BufferPool struct {
	sync.Pool

	Secure bool // if set true, the buffer created will be reset, when putting it back to BufferPool
}

func NewBufferPool(secure bool) *BufferPool {
	bp := &BufferPool{}
	newf := func() any {
		return make([]byte, MAXBUFFERSIZE)
	}
	bp.Pool = sync.Pool{
		New: newf,
	}

	bp.Secure = secure
	return bp
}

func (bp *BufferPool) GetBuffer() []byte { return bp.Get().([]byte) }

func (bp *BufferPool) PutBuffer(p []byte) {
	if bp.Secure {
		for i := range p {
			p[i] = 0
		}
	}
	bp.Put(p)
}
