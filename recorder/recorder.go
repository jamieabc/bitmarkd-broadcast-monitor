package recorder

import (
	"fmt"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

// Recorder - interface for recording records
type Recorder interface {
	Adder
	PeriodicRemover
	Summarizer
}

// Adder - interface for adding a record
type Adder interface {
	Add(time.Time, ...interface{})
}

// PeriodicRemover - interface for periodically removing outdated records
type PeriodicRemover interface {
	PeriodicRemove(clock clock.Clock)
}

// Summarizer - interface for summarizing status of records
type Summarizer interface {
	Summary() SummaryOutput
}

// SummaryOutput - interface for summary output
type SummaryOutput interface {
	fmt.Stringer
	Validator
}

// Validator - interface for deciding if summary output needs to notify
type Validator interface {
	Validate() bool
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
