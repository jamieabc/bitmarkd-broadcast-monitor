package recorder

import (
	"sync"
	"time"
)

type transaction struct{}

type transactions struct {
	sync.Mutex
	data       [recordSize]transaction
	nextItemID int
}

func (t *transactions) Add(time.Time, ...interface{}) {
	return
}

func (t *transactions) Summary() interface{} {
	return nil
}

func newTransactions() Recorder {
	return &transactions{}
}
