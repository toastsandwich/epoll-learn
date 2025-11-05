package pool

import "sync"

type BufferPool struct {
	sync.Pool
	secure bool
}

func NewBufferPool(secure bool) *BufferPool {
	const MAXBUFFERSIZE = 8192
	newF := func() any {
		return make([]byte, MAXBUFFERSIZE)
	}
	return &BufferPool{
		Pool: sync.Pool{
			New: newF,
		},
		secure: secure,
	}
}

func (b *BufferPool) PutBuffer(p []byte) {
	if b.secure {
		for i := range p {
			p[i] = 0
		}
	}
	b.Put(p)
}

func (b *BufferPool) GetBuffer() []byte {
	return b.Get().([]byte)
}
