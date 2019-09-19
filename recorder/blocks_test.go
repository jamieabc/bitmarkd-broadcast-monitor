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
	s := b.Summary().(*recorder.BlocksSummary)

	assert.Equal(t, time.Duration(0), s.Duration, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
}

func TestBlocksSummaryWhenNoCycle(t *testing.T) {
	now := time.Now()
	duration := 5 * time.Second
	b := recorder.NewBlock()
	b.Add(now.Add(-2*duration), recorder.BlockData{
		Hash:   "9999999",
		Number: uint64(1000),
	})
	b.Add(now.Add(-1*duration), recorder.BlockData{
		Hash:   "12345678",
		Number: uint64(1001),
	})
	s := b.Summary().(*recorder.BlocksSummary)

	assert.True(t, s.Duration >= duration, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
	assert.Equal(t, uint64(2), s.BlockCount, "wrong BlockData count")
}

func TestBlocksSummaryWhenCycle(t *testing.T) {
	now := time.Now()
	b := recorder.NewBlock()
	count := 300
	for i := 0; i < count; i++ {
		b.Add(now.Add(time.Duration(-1*count+i)*time.Second), recorder.BlockData{
			Hash:   strconv.Itoa(i),
			Number: uint64(i),
		})
	}
	s := b.Summary().(*recorder.BlocksSummary)

	assert.True(
		t,
		s.Duration >= 100*time.Second,
		"wrong duration",
	)
	assert.Equal(t, 0, len(s.Forks), "wrong fork count")
	assert.Equal(t, uint64(100), s.BlockCount, "wrong BlockData count")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenNoExpiration(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlock()
	duration := 10 * time.Minute

	b.Add(now.Add(-1*duration), recorder.BlockData{
		Hash:   "123456",
		Number: uint64(1000),
	})
	b.Add(now.Add(-5*time.Minute), recorder.BlockData{
		Hash:   "654321",
		Number: uint64(1001),
	})
	s := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= duration, "wrong duration")

	go b.PeriodicRemove(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	s = b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= duration, "wrong duration")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenOneExpiration(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlock()
	b.Add(now.Add(-3*time.Hour), recorder.BlockData{
		Hash:   "123456",
		Number: uint64(1000),
	})
	b.Add(now.Add(-2*time.Second), recorder.BlockData{
		Hash:   "654321",
		Number: uint64(1001),
	})
	s := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= 2*time.Hour, "wrong duration")

	go b.PeriodicRemove(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	s = b.Summary().(*recorder.BlocksSummary)
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
		b.Add(now.Add(time.Duration(-1*i)*time.Minute), recorder.BlockData{
			Hash:   strconv.Itoa(i),
			Number: uint64(i),
		})
	}
	s := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= 3*time.Hour, "wrong duration")

	go b.PeriodicRemove(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	s = b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration <= 2*time.Hour, "wrong larger duration")
	assert.True(t, s.Duration >= 119*time.Minute, "wrong smaller duration")
}

func TestSummaryWhenForkSingleBlock(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)
	b.Add(now, recorder.BlockData{
		Hash:   "123456",
		Number: blockNumber,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "654321",
		Number: blockNumber,
	})

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber, summary.Forks[0].End, "wrong fork end")
}

func TestSummaryWhenForkMultipleBlocks(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)

	b.Add(now, recorder.BlockData{
		Hash:   "1",
		Number: blockNumber,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "2",
		Number: blockNumber + 1,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "3",
		Number: blockNumber + 2,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "4",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "5",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "6",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "7",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "8",
		Number: blockNumber + 5,
	})

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong fork start")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong fork end")
}

func TestSummaryWhenForkMultipleBlocksMultipleTimes(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)

	b.Add(now, recorder.BlockData{
		Hash:   "1",
		Number: blockNumber,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "2",
		Number: blockNumber + 1,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "3",
		Number: blockNumber + 2,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "4",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "5",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "6",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "7",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "8",
		Number: blockNumber + 5,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "9",
		Number: blockNumber + 6,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "10",
		Number: blockNumber + 7,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "11",
		Number: blockNumber + 6,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "12",
		Number: blockNumber + 7,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "13",
		Number: blockNumber + 7,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "14",
		Number: blockNumber + 8,
	})

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 3, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong first fork begin")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong first fork begin")
	assert.Equal(t, blockNumber+6, summary.Forks[1].Begin, "wrong second fork end")
	assert.Equal(t, blockNumber+7, summary.Forks[1].End, "wrong second fork begin")
	assert.Equal(t, blockNumber+7, summary.Forks[2].Begin, "wrong third fork begin")
	assert.Equal(t, blockNumber+7, summary.Forks[2].End, "wrong third fork end")
}

func TestSummaryWhenForkInProgressAndDropOneBlock(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)

	b.Add(now, recorder.BlockData{
		Hash:   "1",
		Number: blockNumber,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "2",
		Number: blockNumber + 1,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "3",
		Number: blockNumber + 2,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "4",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "5",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "6",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "8",
		Number: blockNumber + 8,
	})

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong fork begin")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong fork end")
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

	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "1",
		Number: blockNumber,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "2",
		Number: blockNumber + 1,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "3",
		Number: blockNumber + 2,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "4",
		Number: blockNumber + 3,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "5",
		Number: blockNumber + 4,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "6",
		Number: blockNumber + 3,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "7",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "8",
		Number: blockNumber + 5,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "9",
		Number: blockNumber + 6,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "10",
		Number: blockNumber + 7,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "11",
		Number: blockNumber + 6,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "12",
		Number: blockNumber + 7,
	})

	go b.PeriodicRemove(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+6, summary.Forks[0].Begin, "wrong fork begin")
	assert.Equal(t, blockNumber+7, summary.Forks[0].End, "wrong fork end")
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

	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "1",
		Number: blockNumber,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "2",
		Number: blockNumber + 1,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "3",
		Number: blockNumber + 2,
	})
	b.Add(threeHourBefore, recorder.BlockData{
		Hash:   "4",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "5",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "6",
		Number: blockNumber + 3,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "7",
		Number: blockNumber + 4,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "8",
		Number: blockNumber + 5,
	})

	go b.PeriodicRemove(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.Equal(t, 1, len(summary.Forks), "wrong fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong fork begin")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong fork end")
}

func TestSummaryWhenLongConfirmTime(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)

	b.Add(now, recorder.BlockData{
		Hash:         "1",
		Number:       blockNumber,
		GenerateTime: now.Add(-2 * time.Hour),
		ReceivedTime: now,
	})
	b.Add(now, recorder.BlockData{
		Hash:         "2",
		Number:       blockNumber + 8,
		GenerateTime: now.Add(-1 * time.Hour),
		ReceivedTime: now,
	})

	summary := b.Summary().(*recorder.BlocksSummary)

	assert.Equal(t, 2, len(summary.LongConfirms), "wrong long confirm count")
	assert.Equal(t, 2*time.Hour, summary.LongConfirms[0].Period, "wrong first long confirm period")
	assert.Equal(t, 1*time.Hour, summary.LongConfirms[1].Period, "wrong second long confirm period")
}

func TestSummaryWhenLongConfirmRecycled(t *testing.T) {
	setupRecorder()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()
	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlock()
	threeHourBefore := now.Add(-3 * time.Hour)
	oneHourBefore := now.Add(-1 * time.Hour)
	blockNumber := uint64(1000)

	b.Add(now, recorder.BlockData{
		Hash:         "1",
		Number:       blockNumber,
		GenerateTime: threeHourBefore,
	})
	b.Add(now, recorder.BlockData{
		Hash:         "2",
		Number:       blockNumber + 1,
		GenerateTime: threeHourBefore,
	})
	b.Add(now, recorder.BlockData{
		Hash:         "3",
		Number:       blockNumber + 2,
		GenerateTime: oneHourBefore,
	})

	go b.PeriodicRemove(mock)
	<-time.After(10 * time.Millisecond)
	shutdownChan <- struct{}{}

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.Equal(t, 1, len(summary.LongConfirms), "wrong long confirm count")
	assert.Equal(t, blockNumber+2, summary.LongConfirms[0].BlockNumber, "wrong long confirm block number")
}
