package server

import "sync"

type ConnJob func()

type JobManager struct {
	N    int
	JobQ chan ConnJob

	wg *sync.WaitGroup
}

func NewJobManager(n int) *JobManager {
	j := &JobManager{
		N:    n,
		JobQ: make(chan ConnJob, 1024),
		wg:   &sync.WaitGroup{},
	}

	for range n {
		j.wg.Go(func() {
			for job := range j.JobQ {
				job()
			}
		})
	}
	return j
}

func (j *JobManager) FlushJobs() {
	for len(j.JobQ) > 0 {
		<-j.JobQ
	}
}

func (j *JobManager) Close() {
	j.FlushJobs()
	close(j.JobQ)
	j.wg.Wait() // wait for in proc jobs to finish
}
