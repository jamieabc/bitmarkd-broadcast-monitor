package recorder

import (
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

//Recorder - recorder interface
type Recorder interface {
	Add(time.Time, ...interface{})
	RemoveOutdatedPeriodically(clock clock.Clock)
	Summary() interface{}
}

type expiredAt time.Time
type receivedAt time.Time

const (
	expiredTimeInterval = 2 * time.Hour
)

var (
	overallEarliestTime time.Time
)

//Initialise
func Initialise(shutdown <-chan struct{}) {
	initialiseTransactions(shutdown)
	overallEarliestTime = time.Now()
}
