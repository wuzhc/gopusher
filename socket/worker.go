package socket

type Task func()

type workerPool struct {
	wg      WaitGroupWrapper
	size    int
	taskCh  chan Task
	closeCh chan bool
}

func NewWorkPool(size int) *workerPool {
	return &workerPool{
		size:    size,
		taskCh:  make(chan Task, size),
		closeCh: make(chan bool),
	}
}

// assign task to worker
func (pool *workerPool) init() {
	if pool.size <= 0 {
		pool.size = 1
	}

	for i := 0; i < pool.size; i++ {
		pool.wg.Wrap(pool.newWorker)
	}
}

func (pool *workerPool) newWorker() {
	for {
		select {
		case <-pool.closeCh:
			return
		case task := <-pool.taskCh:
			task()
		}
	}
}

func (pool *workerPool) release() {
	close(pool.closeCh)
	pool.wg.Wait()
}
