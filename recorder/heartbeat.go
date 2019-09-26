package recorder

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/tasks"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

type heartbeat struct {
	sync.Mutex
	data     map[receivedAt]expiredAt
	earliest time.Time
	received bool
}

//HeartbeatSummary - summary of heartbeat data
type HeartbeatSummary struct {
	Duration      time.Duration
	ReceivedCount uint16
	received      bool
	Droprate      float64
}

const (
	maxReceivedCount = 120 / 1 //bitmarkd sends heartbeat every 1 minute, max 120 minutes data preserved
)

var (
	fullCycleReceivedCount float64
	intervalSecond         float64
)

func (h *HeartbeatSummary) String() string {
	if !h.received {
		return neverReceive(h)
	}

	if h.received && 0 == h.ReceivedCount {
		return notReceivingForTwoHours(h)
	}

	dropPercent := math.Floor(h.Droprate*10000) / 100
	return fmt.Sprintf("earliest received time to now takes %s, received: %d, drop percent: %f%%", h.Duration, h.ReceivedCount, dropPercent)
}

func (h *HeartbeatSummary) Valid() bool {
	return true
}

func neverReceive(h *HeartbeatSummary) string {
	expectedCount := math.Floor(h.Duration.Seconds() / intervalSecond)
	if maxReceivedCount < expectedCount {
		expectedCount = maxReceivedCount
	}
	return fmt.Sprintf("not receiving heartbeat for %s, expect to receive %d, drop rate 100%%", h.Duration, int(expectedCount))
}

func notReceivingForTwoHours(h *HeartbeatSummary) string {
	expectedCount := int(math.Floor(h.Duration.Seconds() / intervalSecond))
	return fmt.Sprintf("not receiving heartbeat for more than 2 hours, expected receive count: %d, drop percent: 100%%", expectedCount)
}

//Add - add received heartbeat record
func (h *heartbeat) Add(t time.Time, args ...interface{}) {
	if !h.received {
		h.received = true
	}

	h.Lock()
	h.data[receivedAt(t)] = expiredAt(t.Add(expiredTimeInterval))
	h.Unlock()
}

//PeriodicRemove - clean expired heartbeat record periodically
func (h *heartbeat) PeriodicRemove(args []interface{}) {
	if 2 != len(args) {
		fmt.Println("heartbeat PeriodicRemove wrong arguments length")
		return
	}
	c := args[0].(clock.Clock)
	shutdown := args[1].(<-chan struct{})
	timer := c.After(expiredTimeInterval)
loop:
	for {
		select {
		case <-shutdown:
			break loop
		case <-timer:
			cleanupExpiredHeartbeat(h)
			timer = c.After(time.Duration(intervalSecond) * time.Second)
		}
	}
	fmt.Println("terminate heartbeat PeriodicRemove")
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
func (h *heartbeat) Summary() SummaryOutput {
	h.Lock()

	count := uint16(len(h.data))
	duration, earliest := durationFromEarliestReceive(h)
	updateOverallEarliestTime(h, earliest)

	h.Unlock()

	return &HeartbeatSummary{
		Duration:      duration,
		ReceivedCount: count,
		received:      h.received,
		Droprate:      h.droprate(count, duration),
	}

}

func updateOverallEarliestTime(h *heartbeat, earliest time.Time) {
	if earliest.Before(h.earliest) {
		h.earliest = earliest
	}
}

func durationFromEarliestReceive(h *heartbeat) (time.Duration, time.Time) {
	earliest := h.earliest
	latest := h.earliest

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

	//in case heartbeat time just arrive after monitor start, causes +1 count and make drop percent < 0
	if 0 == expectedReceivedCount || float64(actualReceived) >= expectedReceivedCount {
		return float64(0)
	}

	result := (expectedReceivedCount - float64(actualReceived)) / expectedReceivedCount
	return result
}

//NewHeartbeat - new heartbeat
func NewHeartbeat(interval float64, t tasks.Tasks, ctx context.Context) Recorder {
	h := &heartbeat{
		data:     make(map[receivedAt]expiredAt),
		earliest: time.Now(),
	}
	fullCycleReceivedCount = math.Floor(expiredTimeInterval.Seconds() / interval)
	intervalSecond = interval
	c := clock.NewClock()
	t.Go(h.PeriodicRemove, c, ctx.Done())
	return h
}
