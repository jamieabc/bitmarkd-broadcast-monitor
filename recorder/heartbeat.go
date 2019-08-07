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

var expectedReceivedCount float64

func (h *HeartbeatSummary) String() string {
	if 0 == h.ReceivedCount {
		return fmt.Sprintf("not receiving heartbeat for %s", h.Duration)
	}
	dropPercent := math.Floor(h.Droprate*10000) / 100
	return fmt.Sprintf("earliest received time to now takes %s, received: %d, drop percent: %f%%", h.Duration, h.ReceivedCount, dropPercent)
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

	duration, earliest := durationFromEarliestReceive(h)
	updateOverallEarliestTime(earliest)

	return &HeartbeatSummary{
		Duration:      duration,
		ReceivedCount: count,
		Droprate:      h.droprate(count),
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
	if now.Sub(latestReceivedTime).Seconds() >= h.intervalSecond {
		return now
	}
	return latestReceivedTime
}

func (h *heartbeat) droprate(actualReceived uint16) float64 {
	if 0 == expectedReceivedCount || expectedReceivedCount == float64(actualReceived) {
		return float64(0)
	}

	result := (expectedReceivedCount - float64(actualReceived)) / expectedReceivedCount
	return result
}

//NewHeartbeat - new heartbeat
func NewHeartbeat(intervalSecond float64, shutdownChan <-chan struct{}) Recorder {
	h := &heartbeat{
		data:           make(map[receivedAt]expiredAt),
		intervalSecond: intervalSecond,
		shutdownChan:   shutdownChan,
	}
	expectedReceivedCount = math.Floor(expiredTimeInterval.Seconds() / intervalSecond)
	c := clock.NewClock()
	go h.CleanupPeriodically(c)
	return h
}
