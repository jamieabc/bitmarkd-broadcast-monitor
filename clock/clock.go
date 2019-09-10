package clock

import "time"

//Clock - clock interface
type Clock interface {
	After(time.Duration) <-chan time.Time
	NewTimer(time.Duration) *time.Timer
}

type clock struct{}

// After - return channel after some time
func (c *clock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// NewTimer - return time.Timer
func (c *clock) NewTimer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}

//NewClock - new clock interface
func NewClock() Clock {
	return &clock{}
}
