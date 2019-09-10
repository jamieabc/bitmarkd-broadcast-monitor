package recorder_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
	"github.com/stretchr/testify/assert"
)

func TestBlocksSummaryWhenEmpty(t *testing.T) {
	b := recorder.NewBlocks()
	s := b.Summary().(recorder.BlocksSummary)

	assert.Equal(t, time.Duration(0), s.Duration, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
}

func TestBlocksSummaryWhenOneRecord(t *testing.T) {
	now := time.Now()
	b := recorder.NewBlocks()
	duration := 5 * time.Second
	b.Add(now.Add(-1*duration), uint64(1000), "12345678")
	s := b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration >= duration, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
}

func TestBlocksSummaryWhenCycleRecords(t *testing.T) {
	now := time.Now()
	b := recorder.NewBlocks()
	count := 300
	for i := 0; i < count; i++ {
		b.Add(now.Add(time.Duration(-1*i)*time.Second), uint64(i), strconv.Itoa(i))
	}
	s := b.Summary().(recorder.BlocksSummary)
	assert.True(
		t,
		s.Duration >= (time.Duration(count-1)*time.Second),
		"wrong duration",
	)
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenNoExpiration(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlocks()
	duration := 10 * time.Minute

	b.Add(now.Add(-1*duration), uint64(1000), "123456")
	b.Add(now.Add(-5*time.Minute), uint64(1001), "654321")
	s := b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration >= duration, "wrong duration")

	go b.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	s = b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration >= duration, "wrong duration")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenOneExpiration(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlocks()
	b.Add(now.Add(-3*time.Hour), uint64(1000), "123456")
	b.Add(now.Add(-2*time.Second), uint64(1001), "654321")
	s := b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration >= 2*time.Hour, "wrong duration")

	go b.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	s = b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration >= 2*time.Second, "wrong smaller duration")
	assert.True(t, s.Duration < 2*time.Hour, "wrong larger duration")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenManyExpiration(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlocks()
	for i := 0; i < 200; i++ {
		b.Add(now.Add(time.Duration(-1*i)*time.Minute), uint64(i), strconv.Itoa(i))
	}
	s := b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration >= 3*time.Hour, "wrong duration")

	go b.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	s = b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration <= 2*time.Hour, "wrong larger duration")
	assert.True(t, s.Duration >= 119*time.Minute, "wrong smaller duration")
}
