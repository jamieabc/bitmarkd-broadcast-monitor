package recorder

import (
	"math"
	"sync"
	"time"
)

type heartbeat struct {
	sync.Mutex
	data           map[receivedAt]expiredAt
	nextItemID     int
	intervalSecond float64
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

func (h *heartbeat) CleanupPeriodically() {
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
func NewHeartbeat(intervalSecond float64) Recorder {
	return &heartbeat{
		data:           make(map[receivedAt]expiredAt),
		intervalSecond: intervalSecond,
	}
}
