package worker

import (
	"fmt"
	"sync"
)

type WorkerPool interface {
	Submit(task func()) error
	Run()
	Shutdown()
}

type GoWorkerPool struct {
	tasks chan func()
	wg    sync.WaitGroup
	quit  chan struct{}
}

// NewGoWorkerPool creates a new GoWorkerPool with the specified number of workers.
// Each worker runs in its own goroutine and listens for tasks to execute from the tasks channel.
// The pool can be stopped by closing the quit channel.
//
// Parameters:
//   workerCount - the number of workers to create in the pool.
//
// Returns:
//   A pointer to the newly created GoWorkerPool.
func NewGoWorkerPool(workerCount int) *GoWorkerPool {
	pool := &GoWorkerPool{
		tasks: make(chan func()),
		quit:  make(chan struct{}),
	}
	for range workerCount {
		pool.wg.Add(1)
		go func() {
			defer pool.wg.Done()
			for {
				select {
				case task := <-pool.tasks:
					task()
				case <-pool.quit:
					return
				}
			}
		}()
	}
	return pool
}

func (wp *GoWorkerPool) Submit(task func()) error {
	select {
	case wp.tasks <- task:
		return nil
	case <-wp.quit:
		return fmt.Errorf("worker pool is shutting down")
	}
}

func (wp *GoWorkerPool) Run() {}

func (wp *GoWorkerPool) Shutdown() {
	close(wp.quit)
	wp.wg.Wait()
}
