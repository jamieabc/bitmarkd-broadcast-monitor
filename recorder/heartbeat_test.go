package recorder_test

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/tasks"

	"github.com/golang/mock/gomock"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"

	"github.com/stretchr/testify/assert"
)

const (
	intervalSecond = 1
)

var (
	fullCycleExpectedReceivedCount = math.Floor(expiredTimeInterval.Seconds() / intervalSecond)
)

var (
	cancel    context.CancelFunc
	ctx       context.Context
	done      chan struct{}
	startTime time.Time
	task      tasks.Tasks
)

func init() {
	startTime = time.Now()
	ctx, cancel = context.WithCancel(context.Background())
	task = tasks.NewTasks(done, cancel)
}

func setupHeartbeat() recorder.Recorder {
	return recorder.NewHeartbeat(intervalSecond, task, ctx)
}

func TestNewHeartbeat(t *testing.T) {
	r := setupHeartbeat()
	now := time.Now()

	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.WithinDuration(t, now, now.Add(summary.Duration), 1*time.Second, "wrong initial heartbeat duration")
	assert.Equal(t, uint16(0), summary.ReceivedCount, "wrong initial heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong initial drop rate")
}

func TestHeartbeatSummaryWhenSingle(t *testing.T) {
	r := setupHeartbeat()
	now := time.Now()
	delaySeconds := 60
	r.Add(now.Add(time.Duration(-1*delaySeconds) * time.Second))
	summary := r.Summary().(*recorder.HeartbeatSummary)
	expectedReceivedCount := float64(delaySeconds-1) / intervalSecond

	assert.WithinDuration(t, now, now.Add(summary.Duration), expiredTimeInterval, "wrong duration")
	assert.Equal(t, uint16(1), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, (expectedReceivedCount-1)/expectedReceivedCount, summary.Droprate, "wrong drop rate")
}

func TestHeartbeatSummaryWhenEdge(t *testing.T) {
	r := setupHeartbeat()
	now := time.Now()
	size := 40
	for i := 0; i < size; i++ {
		r.Add(now.Add(time.Duration(-1*i) * time.Second))
	}
	summary := r.Summary().(*recorder.HeartbeatSummary)
	duration := now.Sub(startTime)
	droprate := float64(0)
	fmt.Printf("duration: %s\n", duration)
	if duration > time.Second {
		droprate = float64(size) / duration.Seconds()
	}

	assert.WithinDuration(t, now, now.Add(time.Duration(size-1)*time.Second), summary.Duration, "wrong duration")
	assert.Equal(t, uint16(size), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, droprate, summary.Droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenDrop(t *testing.T) {
	r := setupHeartbeat()
	now := time.Now()
	size := 20
	for i := 0; i < size; i++ {
		if 0 < i && 0 == i%5 {
			continue
		}
		r.Add(now.Add(time.Duration(-1*i) * time.Second))
	}
	summary := r.Summary().(*recorder.HeartbeatSummary)
	expectedReceivedCount := float64((size - 1) / intervalSecond)

	assert.WithinDuration(t, now, now.Add(time.Duration(size-1)*time.Second), summary.Duration, "wrong duration")
	assert.Equal(t, uint16(size-3), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, (expectedReceivedCount-float64(summary.ReceivedCount))/expectedReceivedCount, summary.Droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenNormal(t *testing.T) {
	r := setupHeartbeat()
	now := time.Now()
	size := 15
	for i := 0; i < size; i++ {
		r.Add(now.Add(time.Duration(-1*i) * time.Second))
	}
	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.WithinDuration(t, now, now.Add(time.Duration(size-1)*time.Second), summary.Duration, "wrong duration")
	assert.Equal(t, uint16(size), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenMore(t *testing.T) {
	r := setupHeartbeat()
	now := time.Now()
	size := 20
	for i := 0; i < size; i++ {
		r.Add(now.Add(time.Duration(-1*i) * time.Second))
	}
	r.Add(now.Add(time.Duration(-1500) * time.Millisecond))
	summary := r.Summary().(*recorder.HeartbeatSummary)

	assert.WithinDuration(t, now, now.Add(time.Duration(size-1)*time.Second), summary.Duration, "wrong duration")
	assert.Equal(t, uint16(size+1), summary.ReceivedCount, "wrong heartbeat count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestHeartbeatRemoveOutdatedPeriodicallyWhenExpiration(t *testing.T) {
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	r := setupHeartbeat()
	now := time.Now()
	r.Add(now.Add(-1 * expiredTimeInterval))
	r.Add(now)
	summary := r.Summary().(*recorder.HeartbeatSummary)
	droprate := (fullCycleExpectedReceivedCount - float64(summary.ReceivedCount)) / fullCycleExpectedReceivedCount

	assert.Equal(t, uint16(2), summary.ReceivedCount, "wrong count")
	assert.Equal(t, droprate, summary.Droprate, "wrong droprate")

	go r.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	summary = r.Summary().(*recorder.HeartbeatSummary)
	assert.Equal(t, uint16(1), summary.ReceivedCount, "wrong count")
	assert.Equal(t, (fullCycleExpectedReceivedCount-float64(1))/fullCycleExpectedReceivedCount, summary.Droprate, "wrong droprate")
}

func TestHeartbeatRemoveOutdatedPeriodicallyWhenNoExpiration(t *testing.T) {
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	r := setupHeartbeat()
	now := time.Now()
	r.Add(now)
	summary := r.Summary().(*recorder.HeartbeatSummary)
	assert.Equal(t, uint16(1), summary.ReceivedCount, "wrong count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")

	go r.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	summary = r.Summary().(*recorder.HeartbeatSummary)
	assert.Equal(t, uint16(1), summary.ReceivedCount, "wrong count")
	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}
