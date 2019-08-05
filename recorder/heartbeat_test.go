package recorder_test

import (
	"testing"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"

	"github.com/stretchr/testify/assert"
)

const (
	heartbeatInterval = 1
)

func TestNewHeartbeat(t *testing.T) {
	r := recorder.NewHeartbeat(heartbeatInterval)

	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.Equal(t, time.Duration(0), summary.Duration, "wrong initial heartbeat duration")
	assert.Equal(t, uint16(0), summary.ReceivedCount, "wrong initial heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong initial drop rate")
}

func TestHeartbeatSummaryWhenSingle(t *testing.T) {
	r := recorder.NewHeartbeat(heartbeatInterval)
	now := time.Now()
	r.Add(now)
	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.Equal(t, time.Duration(0), summary.Duration, "wrong duration")
	assert.Equal(t, uint16(1), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong drop rate")
}

func TestHeartbeatSummaryWhenEdge(t *testing.T) {
	r := recorder.NewHeartbeat(heartbeatInterval)
	now := time.Now()
	size := 40
	for i := 0; i < size; i++ {
		r.Add(now.Add(time.Duration(i) * time.Second))
	}
	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.Equal(t, time.Duration(39)*time.Second, summary.Duration, "wrong duration")
	assert.Equal(t, uint16(40), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenCycle(t *testing.T) {
	r := recorder.NewHeartbeat(heartbeatInterval)
	now := time.Now()
	size := 100
	for i := 0; i < size; i++ {
		r.Add(now.Add(time.Duration(i) * time.Second))
	}
	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.Equal(t, time.Duration(size-1)*time.Second, summary.Duration, "wrong duration")
	assert.Equal(t, uint16(size), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenDrop(t *testing.T) {
	r := recorder.NewHeartbeat(heartbeatInterval)
	now := time.Now()
	size := 20
	for i := 0; i < size; i++ {
		if 0 < i && 0 == i%5 {
			continue
		}
		r.Add(now.Add(time.Duration(i) * time.Second))
	}
	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.Equal(t, time.Duration(size-1)*time.Second, summary.Duration, "wrong duration")
	assert.Equal(t, uint16(size-3), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, float64(3*100/size), summary.Droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenEnough(t *testing.T) {
	r := recorder.NewHeartbeat(heartbeatInterval)
	now := time.Now()
	size := 15
	for i := 0; i < size; i++ {
		r.Add(now.Add(time.Duration(i) * time.Second))
	}
	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.Equal(t, time.Duration(14)*time.Second, summary.Duration, "wrong duration")
	assert.Equal(t, uint16(15), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}
