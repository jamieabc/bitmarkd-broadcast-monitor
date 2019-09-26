package tasks

import (
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"
)

type task struct {
	cancel  context.CancelFunc
	done    chan<- struct{}
	wg      sync.WaitGroup
	counter uint64
}

// Go - invoke a goroutine to run
// execution function needs to receive context.Done signal
func (t *task) Go(exec executionFunction, args ...interface{}) {
	go func(t *task) {
		t.wg.Add(1)
		atomic.AddUint64(&t.counter, 1)
		exec(args)
		atomic.AddUint64(&t.counter, ^uint64(0))
		t.wg.Done()
		fmt.Printf("counter: %d\n", t.counter)
	}(t)
}

// Done - stop a goroutine
func (t *task) Done() {
	t.cancel()
	t.wg.Wait()
	t.done <- struct{}{}
}

// NewTasks - create instance for Tasks interface
// done is used to notify that all goroutines are terminated
// cancel is used to send signal to all running goroutines
func NewTasks(done chan<- struct{}, cancel context.CancelFunc) Tasks {
	return &task{
		cancel: cancel,
		done:   done,
	}
}
