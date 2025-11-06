package pool

import "sync"

// type BufferPool struct {
// 	sync.Pool
// 	secure bool
// }

const MAXBUFFERSIZE = 8192

var BufferPool = sync.Pool{
	New: func() any {
		return make([]byte, MAXBUFFERSIZE)
	},
}

func GetBuffer() []byte {
	return BufferPool.Get().([]byte)
}

func PutBuffer(p []byte) {
	for i := range len(p) {
		p[i] = 0
	}
	BufferPool.Put(p)
}
