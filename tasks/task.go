package tasks

import (
	"sync"

	"golang.org/x/net/context"
)

type task struct {
	cancel context.CancelFunc
	done   chan<- struct{}
	wg     sync.WaitGroup
}

// Go - invoke a goroutine to run
// execution function needs to receive context.Done signal
func (t *task) Go(exec executionFunction, args ...interface{}) {
	go func(t *task) {
		t.wg.Add(1)
		exec(args)
		t.wg.Done()
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
