package worker

import (
    "fmt"
    "log"
    "sync"
    "time"
)

type WorkerPool interface {
    Submit(task func()) error
    Run()
    Shutdown()
}

type GoWorkerPool struct {
    tasks       chan func()
    wg          sync.WaitGroup
    quit        chan struct{}
    workerCount int
    running     bool
    mu          sync.Mutex // Protects running state
}

// NewGoWorkerPool creates a new GoWorkerPool with the specified number of workers.
// The pool is initialized but not started until Run() is called.
//
// Parameters:
//   workerCount - the number of workers to create in the pool.
//
// Returns:
//   A pointer to the newly created GoWorkerPool.
func NewGoWorkerPool(workerCount int) *GoWorkerPool {
    if workerCount <= 0 {
        workerCount = 1 // Ensure at least one worker
    }
    
    return &GoWorkerPool{
        tasks:       make(chan func(), workerCount*10),
        quit:        make(chan struct{}),
        workerCount: workerCount,
        running:     false,
    }
}

func (wp *GoWorkerPool) worker(id int) {
    defer wp.wg.Done()
    
    log.Printf("Worker %d started", id)
    
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Worker %d recovered from panic: %v", id, r)
        }
        log.Printf("Worker %d stopped", id)
    }()
    
    for {
        select {
        case task, ok := <-wp.tasks:
            if !ok {
                return
            }
            
            func() {
                defer func() {
                    if r := recover(); r != nil {
                        log.Printf("Task panicked in worker %d: %v", id, r)
                    }
                }()
                
                task()
            }()
            
        case <-wp.quit:
            return
        }
    }
}

func (wp *GoWorkerPool) Submit(task func()) error {
    if task == nil {
        return fmt.Errorf("cannot submit nil task")
    }
    
    wp.mu.Lock()
    if !wp.running {
        wp.mu.Unlock()
        return fmt.Errorf("worker pool is not running")
    }
    wp.mu.Unlock()
    
    select {
    case wp.tasks <- task:
        return nil
    case <-wp.quit:
        return fmt.Errorf("worker pool is shutting down")
    case <-time.After(100 * time.Millisecond):
        return fmt.Errorf("worker pool queue is full")
    }
}

// Run starts the worker pool.
// If the pool is already running, this is a no-op.
func (wp *GoWorkerPool) Run() {
    wp.mu.Lock()
    defer wp.mu.Unlock()
    
    if wp.running {
        return
    }
    
    log.Printf("Starting worker pool with %d workers", wp.workerCount)
    
    for i := 0; i < wp.workerCount; i++ {
        wp.wg.Add(1)
        go wp.worker(i)
    }
    
    wp.running = true
}

// Shutdown stops the worker pool and waits for all workers to complete.
// Any tasks still in the queue will not be processed.
func (wp *GoWorkerPool) Shutdown() {
    wp.mu.Lock()
    if !wp.running {
        wp.mu.Unlock()
        return
    }
    wp.running = false
    wp.mu.Unlock()
    
    log.Printf("Shutting down worker pool")
    close(wp.quit)
    
    done := make(chan struct{})
    go func() {
        wp.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        log.Printf("Worker pool shutdown complete")
    case <-time.After(5 * time.Second):
        log.Printf("Worker pool shutdown timed out after 5 seconds")
    }
}

func (wp *GoWorkerPool) ActiveWorkerCount() int {
    return wp.workerCount
}

func (wp *GoWorkerPool) IsRunning() bool {
    wp.mu.Lock()
    defer wp.mu.Unlock()
    return wp.running
}
