package recorder

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

type heartbeat struct {
	sync.Mutex
	data         map[receivedAt]expiredAt
	nextItemID   int
	received     bool
	shutdownChan <-chan struct{}
}

//HeartbeatSummary - summary of heartbeat data
type HeartbeatSummary struct {
	Duration      time.Duration
	ReceivedCount uint16
	received      bool
	Droprate      float64
}

var (
	fullCycleReceivedCount float64
	intervalSecond         float64
)

func (h *HeartbeatSummary) String() string {
	if !h.received {
		expectedCount := math.Floor(h.Duration.Seconds() / intervalSecond)
		return fmt.Sprintf("not receiving heartbeat for %s, expected received %d", h.Duration, int(expectedCount))
	}
	dropPercent := math.Floor(h.Droprate*10000) / 100
	return fmt.Sprintf("earliest received time to now takes %s, received: %d, drop percent: %f%%", h.Duration, h.ReceivedCount, dropPercent)
}

//Add - add received heartbeat record
func (h *heartbeat) Add(t time.Time, args ...interface{}) {
	h.received = true

	h.Lock()
	h.data[receivedAt(t)] = expiredAt(t.Add(expiredTimeInterval))
	h.Unlock()
}

//RemoveOutdatedPeriodically - clean expired heartbeat record periodically
func (h *heartbeat) RemoveOutdatedPeriodically(c clock.Clock) {
	timer := c.After(expiredTimeInterval)
loop:
	for {
		select {
		case <-h.shutdownChan:
			break loop
		case <-timer:
			cleanupExpiredHeartbeat(h)
			timer = c.After(time.Duration(intervalSecond) * time.Second)
		}
	}
}

func cleanupExpiredHeartbeat(h *heartbeat) {
	now := time.Now()

	h.Lock()

	for k, v := range h.data {
		if now.After(time.Time(v)) {
			delete(h.data, k)
		}
	}

	h.Unlock()
}

//Summary - summarize heartbeat data
func (h *heartbeat) Summary() interface{} {
	h.Lock()

	count := uint16(len(h.data))
	duration, earliest := durationFromEarliestReceive(h)
	updateOverallEarliestTime(earliest)

	h.Unlock()

	return &HeartbeatSummary{
		Duration:      duration,
		ReceivedCount: count,
		received:      h.received,
		Droprate:      h.droprate(count, duration),
	}

}

func updateOverallEarliestTime(earliest time.Time) {
	if earliest.Before(overallEarliestTime) {
		overallEarliestTime = earliest
	}
}

func durationFromEarliestReceive(h *heartbeat) (time.Duration, time.Time) {
	earliest := overallEarliestTime
	latest := time.Time{}
	for k := range h.data {
		if earliest.After(time.Time(k)) {
			earliest = time.Time(k)
		}
		if latest.Before(time.Time(k)) {
			latest = time.Time(k)
		}
	}
	actualLatest := h.chooseClosestLatestReceiveTime(latest)
	return actualLatest.Sub(earliest), earliest
}

func (h *heartbeat) chooseClosestLatestReceiveTime(latestReceivedTime time.Time) time.Time {
	now := time.Now()
	if now.Sub(latestReceivedTime).Seconds() >= intervalSecond {
		return now
	}
	return latestReceivedTime
}

func (h *heartbeat) droprate(actualReceived uint16, duration time.Duration) float64 {
	if 0 == actualReceived {
		return float64(0)
	}
	expectedReceivedCount := fullCycleReceivedCount
	if duration < expiredTimeInterval {
		expectedReceivedCount = math.Floor(duration.Seconds() / intervalSecond)
	}
	if 0 == expectedReceivedCount {
		return float64(0)
	}

	result := (expectedReceivedCount - float64(actualReceived)) / expectedReceivedCount
	return result
}

//NewHeartbeat - new heartbeat
func NewHeartbeat(interval float64, shutdownChan <-chan struct{}) Recorder {
	h := &heartbeat{
		data:         make(map[receivedAt]expiredAt),
		shutdownChan: shutdownChan,
	}
	fullCycleReceivedCount = math.Floor(expiredTimeInterval.Seconds() / interval)
	intervalSecond = interval
	c := clock.NewClock()
	go h.RemoveOutdatedPeriodically(c)
	return h
}
