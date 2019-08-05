package recorder

import (
	"math"
	"sync"
	"time"
)

type heartbeat struct {
	sync.Mutex
	data           [recordSize]time.Time
	nextItemID     int
	intervalSecond float64
}

//HeartbeatSummary - summary of heartbeat data
type HeartbeatSummary struct {
	Duration      time.Duration
	ReceivedCount uint16
	Droprate      float64
}

const (
	recordSize = 90
)

//Add - add received heartbeat record
func (h *heartbeat) Add(t time.Time, args ...interface{}) {
	h.Lock()
	defer h.Unlock()

	h.data[h.nextItemID] = t
	h.nextItemID = nextID(h.nextItemID)
}

//Summary -summarize heartbeat data
func (h *heartbeat) Summary() interface{} {
	h.Lock()
	defer h.Unlock()

	min := h.data[0]
	count := uint16(0)

	for i := 0; i < recordSize; i++ {
		if (time.Time{}) != h.data[i] {
			count++
			if h.data[i].Before(min) {
				min = h.data[i]
			}
		} else {
			break
		}
	}

	latestReceiveTime := h.data[prevID(h.nextItemID)]

	if (time.Time{}) == min || latestReceiveTime.Before(min) {
		return &HeartbeatSummary{
			Duration:      0,
			ReceivedCount: count,
			Droprate:      0,
		}
	}

	duration := latestReceiveTime.Sub(min)

	return &HeartbeatSummary{
		Duration:      duration,
		ReceivedCount: count,
		Droprate:      h.droprate(duration, count),
	}
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
		intervalSecond: intervalSecond,
	}
}
