// Package concurrent provides concurrency utilities including goroutine pools.
package concurrent

import (
	"context"
	"sync"
)

// Pool manages a pool of workers for concurrent task execution.
type Pool struct {
	size   int
	tasks  chan func()
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPool creates a new goroutine pool with the specified number of workers.
func NewPool(size int) *Pool {
	if size <= 0 {
		size = 4
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		size:   size,
		tasks:  make(chan func(), size*2),
		ctx:    ctx,
		cancel: cancel,
	}
	pool.start()
	return pool
}

func (p *Pool) start() {
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Recover from panics in tasks
						_ = r
					}
				}()
				task()
			}()
		}
	}
}

// Submit adds a task to the pool for execution.
func (p *Pool) Submit(task func()) {
	select {
	case p.tasks <- task:
	default:
		// If the channel is full, execute synchronously
		task()
	}
}

// SubmitWithContext adds a task that respects context cancellation.
func (p *Pool) SubmitWithContext(ctx context.Context, task func()) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	p.Submit(func() {
		select {
		case <-ctx.Done():
			return
		default:
			task()
		}
	})
}

// Stop gracefully stops the pool, waiting for all tasks to complete.
func (p *Pool) Stop() {
	close(p.tasks)
	p.wg.Wait()
	p.cancel()
}

// Size returns the number of workers in the pool.
func (p *Pool) Size() int {
	return p.size
}
