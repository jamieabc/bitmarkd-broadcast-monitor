package clock

import "time"

//Clock - clock interface
type Clock interface {
	After(time.Duration) <-chan time.Time
}

type clock struct{}

//After - return channel after some time
func (c *clock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

//NewClock - new clock interface
func NewClock() Clock {
	return &clock{}
}
