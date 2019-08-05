package recorder

import (
	"math"
	"sync"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

type heartbeat struct {
	sync.Mutex
	data           map[receivedAt]expiredAt
	nextItemID     int
	intervalSecond float64
	shutdownChan   <-chan struct{}
}

//HeartbeatSummary - summary of heartbeat data
type HeartbeatSummary struct {
	Duration      time.Duration
	ReceivedCount uint16
	Droprate      float64
}

//Add - add received heartbeat record
func (h *heartbeat) Add(t time.Time, args ...interface{}) {
	h.Lock()
	defer h.Unlock()

	h.data[receivedAt(t)] = expiredAt(t.Add(expiredTimeInterval))
}

//CleanupPeriodically - clean expired heartbeat record periodically
func (h *heartbeat) CleanupPeriodically(c clock.Clock) {
	timer := c.After(expiredTimeInterval)
loop:
	for {
		select {
		case <-h.shutdownChan:
			break loop
		case <-timer:
			cleanupExpiredHeartbeat(h)
			timer = c.After(time.Duration(h.intervalSecond) * time.Second)
		}
	}
}

func cleanupExpiredHeartbeat(h *heartbeat) {
	now := time.Now()
	h.Lock()
	defer h.Unlock()

	for k, v := range h.data {
		if now.After(time.Time(v)) {
			delete(h.data, k)
		}
	}
}

//Summary - summarize heartbeat data
func (h *heartbeat) Summary() interface{} {
	h.Lock()
	defer h.Unlock()

	count := uint16(len(h.data))
	if 0 == count {
		return &HeartbeatSummary{
			Duration:      0,
			ReceivedCount: 0,
			Droprate:      0,
		}
	}

	earliest, latest := findEarliestAndLatest(h.data)
	duration := time.Time(latest).Sub(time.Time(earliest))

	return &HeartbeatSummary{
		Duration:      duration,
		ReceivedCount: uint16(len(h.data)),
		Droprate:      h.droprate(duration, count),
	}
}

func findEarliestAndLatest(data map[receivedAt]expiredAt) (expiredAt, expiredAt) {
	earliest := expiredAt(time.Time{})
	latest := expiredAt(time.Time{})
	for _, v := range data {
		if expiredAt(time.Time{}) == earliest || time.Time(earliest).After(time.Time(v)) {
			earliest = v
		}

		if expiredAt(time.Time{}) == latest || time.Time(latest).Before(time.Time(v)) {
			latest = v
		}
	}
	return earliest, latest
}

func (h *heartbeat) droprate(duration time.Duration, actualReceived uint16) float64 {
	expectedCount := math.Floor(duration.Seconds()/h.intervalSecond) + 1
	if 0 == expectedCount {
		return float64(0)
	}

	return (expectedCount - float64(actualReceived)) * 100 / expectedCount
}

//NewHeartbeat - new heartbeat
func NewHeartbeat(intervalSecond float64, shutdownChan <-chan struct{}) Recorder {
	h := &heartbeat{
		data:           make(map[receivedAt]expiredAt),
		intervalSecond: intervalSecond,
		shutdownChan:   shutdownChan,
	}
	c := clock.NewClock()
	go h.CleanupPeriodically(c)
	return h
}
