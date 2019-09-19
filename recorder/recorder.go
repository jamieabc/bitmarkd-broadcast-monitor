package recorder

import (
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

// Recorder - interface for recording records
type Recorder interface {
	Adder
	PeriodicRemover
	Summarizer
}

// Adder - interface for adding record
type Adder interface {
	Add(time.Time, ...interface{})
}

// PeriodicRemover - interface for removing outdated record
type PeriodicRemover interface {
	PeriodicRemove(clock clock.Clock)
}

// Summarizer - interface for summarizing records
type Summarizer interface {
	Summary() interface{}
}

type expiredAt time.Time
type receivedAt time.Time

const (
	expiredTimeInterval = 2 * time.Hour
	totalReceivedCount  = int(expiredTimeInterval / time.Minute)
	indexNotFound       = -1
)

var (
	overallEarliestTime time.Time
)

// Initialise
func Initialise(shutdown <-chan struct{}) {
	initialiseTransactions(shutdown)
	overallEarliestTime = time.Now()
}
