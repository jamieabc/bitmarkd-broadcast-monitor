package tasks_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/tasks"
)

func TestDone(t *testing.T) {
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	myTask := tasks.NewTasks(done, cancel)
	var wgGoroutineReady sync.WaitGroup
	wgGoroutineReady.Add(2)

	myTask.Go(func(args []interface{}) {
		wgGoroutineReady.Done()
		select {
		case <-ctx.Done():
			return
		}
	})

	myTask.Go(func(args []interface{}) {
		wgGoroutineReady.Done()
		select {
		case <-ctx.Done():
			return
		}
	})
	wgGoroutineReady.Wait()

	channelStartWaiting := make(chan struct{})
	channelReceivedSignal := make(chan struct{})

	go func(ch <-chan struct{}) {
		channelStartWaiting <- struct{}{}
		<-ch
		channelReceivedSignal <- struct{}{}
	}(done)

	<-channelStartWaiting
	myTask.Done()
	<-channelReceivedSignal
}
