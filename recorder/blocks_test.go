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
	b := recorder.NewBlock()
	s := b.Summary().(recorder.BlocksSummary)

	assert.Equal(t, time.Duration(0), s.Duration, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
}

func TestBlocksSummaryWhenOneRecord(t *testing.T) {
	now := time.Now()
	b := recorder.NewBlock()
	duration := 5 * time.Second
	b.Add(now.Add(-1*duration), uint64(1000), "12345678")
	s := b.Summary().(recorder.BlocksSummary)
	assert.True(t, s.Duration >= duration, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
}

func TestBlocksSummaryWhenCycleRecords(t *testing.T) {
	now := time.Now()
	b := recorder.NewBlock()
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
	b := recorder.NewBlock()
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
	b := recorder.NewBlock()
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
	b := recorder.NewBlock()
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

func TestSummaryWhenForkSingleBlock(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)
	b.Add(now, blockNumber, "123456")
	b.Add(now, blockNumber, "654321")

	summary := b.Summary().(recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber, summary.Forks[0].End, "wrong fork end")
}

func TestSummaryWhenForkMultipleBlocks(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)
	b.Add(now, blockNumber, "1")
	b.Add(now, blockNumber+1, "2")
	b.Add(now, blockNumber+2, "3")
	b.Add(now, blockNumber+3, "4")
	b.Add(now, blockNumber+4, "5")
	b.Add(now, blockNumber+3, "6")
	b.Add(now, blockNumber+4, "7")
	b.Add(now, blockNumber+5, "8")

	summary := b.Summary().(recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong fork end")
}

func TestSummaryWhenForkMultipleBlocksMultipleTimes(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)
	b.Add(now, blockNumber, "1")
	b.Add(now, blockNumber+1, "2")
	b.Add(now, blockNumber+2, "3")
	b.Add(now, blockNumber+3, "4")
	b.Add(now, blockNumber+4, "5")
	b.Add(now, blockNumber+3, "6")
	b.Add(now, blockNumber+4, "7")
	b.Add(now, blockNumber+5, "8")
	b.Add(now, blockNumber+6, "9")
	b.Add(now, blockNumber+7, "10")
	b.Add(now, blockNumber+6, "11")
	b.Add(now, blockNumber+7, "12")

	summary := b.Summary().(recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 2, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong fork start")
	assert.Equal(t, blockNumber+6, summary.Forks[1].Begin, "wrong fork end")
	assert.Equal(t, blockNumber+7, summary.Forks[1].End, "wrong fork start")
}

func TestSummaryWhenForkInProgressAndDropOneBlock(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)
	b.Add(now, blockNumber, "1")
	b.Add(now, blockNumber+1, "2")
	b.Add(now, blockNumber+2, "3")
	b.Add(now, blockNumber+3, "4")
	b.Add(now, blockNumber+4, "5")
	b.Add(now, blockNumber+3, "6")
	b.Add(now, blockNumber+5, "8")

	summary := b.Summary().(recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong fork start")
}

func TestSummaryWhenForkBlockRecycledEntirely(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()
	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	b := recorder.NewBlock()
	now := time.Now()
	threeHourBefore := now.Add(-3 * time.Hour)
	blockNumber := uint64(1000)

	b.Add(threeHourBefore, blockNumber, "1")
	b.Add(threeHourBefore, blockNumber+1, "2")
	b.Add(threeHourBefore, blockNumber+2, "3")
	b.Add(threeHourBefore, blockNumber+3, "4")
	b.Add(threeHourBefore, blockNumber+4, "5")
	b.Add(threeHourBefore, blockNumber+3, "6")
	b.Add(threeHourBefore, blockNumber+4, "5")
	b.Add(now, blockNumber+5, "8")
	b.Add(now, blockNumber+6, "9")
	b.Add(now, blockNumber+7, "10")
	b.Add(now, blockNumber+6, "11")
	b.Add(now, blockNumber+7, "12")

	go b.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	summary := b.Summary().(recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+6, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber+7, summary.Forks[0].End, "wrong fork start")
}

func TestSummaryWhenForkBlockRecycledPartially(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()
	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlock()
	threeHourBefore := now.Add(-3 * time.Hour)
	blockNumber := uint64(1000)

	b.Add(threeHourBefore, blockNumber, "1")
	b.Add(threeHourBefore, blockNumber+1, "2")
	b.Add(threeHourBefore, blockNumber+2, "3")
	b.Add(threeHourBefore, blockNumber+3, "4")
	b.Add(now, blockNumber+4, "5")
	b.Add(now, blockNumber+3, "6")
	b.Add(now, blockNumber+4, "5")
	b.Add(now, blockNumber+5, "8")

	go b.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	summary := b.Summary().(recorder.BlocksSummary)
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong fork start")
}
