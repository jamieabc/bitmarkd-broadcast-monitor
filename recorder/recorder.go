package recorder

import (
	"time"
)

//Recorder - recorder interface
type Recorder interface {
	Add(time.Time, ...interface{})
	CleanupPeriodically()
	Summary() interface{}
}

type expiredAt time.Time
type receivedAt time.Time

const (
	expiredTimeInterval = 2 * time.Hour
)
