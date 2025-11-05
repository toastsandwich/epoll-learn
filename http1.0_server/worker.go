package server

import (
	"fmt"
	"sync"
)

type Job struct {
	Fd   int
	Data []byte
}

type Result struct {
	Fd   int
	Data []byte
}

var MasterRecords = NewRecords()

func NewRecords() *Records {
	return &Records{
		Data: make(map[int][]byte),
	}
}

type Data struct {
	EpollFd int
	P       []byte
}

type Records struct {
	Data map[int][]byte // int is fd
	mu   sync.Mutex
}

func (r *Records) Add(fd int, p []byte) {
	r.mu.Lock()
	r.Data[fd] = p
	r.mu.Unlock()
}

// Submit will be used when we get EPOLLOUT event. submit return value for the respective fd and delete it from record.
func (r *Records) Submit(fd int) []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.Data[fd]; ok {
		delete(r.Data, fd)
		return p
	}
	return nil
}

type JobManager struct {
	N int

	In  chan Job
	Out chan Result

	wg *sync.WaitGroup
}

func NewJobManager(maxworkercount int) *JobManager {
	j := &JobManager{
		N:   maxworkercount,
		In:  make(chan Job, 1024),
		Out: make(chan Result, 1024),
		wg:  &sync.WaitGroup{},
	}

	// start workers now
	for range j.N {
		j.wg.Go(func() {
			for job := range j.In {
				req, err := parseRequest(job.Data)
				if err != nil {
					fmt.Println("error parsing request", err)
					continue
				}
				fmt.Println(req)
				j.toOutChan(job.Fd, []byte("response"))
			}
		})
	}

	j.wg.Go(func() {
		for res := range j.Out {
			MasterRecords.Add(res.Fd, res.Data)
		}
	})

	return j
}

func (j *JobManager) toInChan(fd int, data []byte) {
	j.In <- Job{fd, data}
}

func (j *JobManager) toOutChan(fd int, data []byte) {
	j.Out <- Result{fd, data}
}

func (j *JobManager) Close() {
	close(j.In)
	j.wg.Wait()
	close(j.Out)
}
