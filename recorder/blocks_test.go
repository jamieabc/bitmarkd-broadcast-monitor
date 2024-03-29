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
	assert.Equal(t, 0, len(s.Forks), "wrong Fork count")
}

func TestBlocksSummaryWhenNoCycle(t *testing.T) {
	now := time.Now()
	duration := 5 * time.Second
	b := recorder.NewBlock()
	b.Add(now.Add(-2*duration), recorder.BlockData{
		Hash:   "1000",
		Number: uint64(1000),
	})
	b.Add(now.Add(-1*duration), recorder.BlockData{
		Hash:   "1001",
		Number: uint64(1001),
	})
	s := b.Summary().(*recorder.BlocksSummary)

	assert.True(t, s.Duration >= duration, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong Fork count")
	assert.Equal(t, uint64(2), s.BlockCount, "wrong BlockData count")
}

func TestBlocksSummaryWhenCycle(t *testing.T) {
	now := time.Now()
	b := recorder.NewBlock()
	count := 300
	for i := 0; i < count; i++ {
		b.Add(now.Add(time.Duration(-1*(count-i))*time.Second), recorder.BlockData{
			Hash:   strconv.Itoa(i),
			Number: uint64(i),
		})
	}
	s := b.Summary().(*recorder.BlocksSummary)

	assert.True(t, s.Duration >= 99*time.Second, "wrong duration")
	assert.Equal(t, 0, len(s.Forks), "wrong Fork count")
	assert.Equal(t, uint64(100), s.BlockCount, "wrong BlockData count")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenNoExpiration(t *testing.T) {
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlock()

	b.Add(now.Add(-10*time.Minute), recorder.BlockData{
		Hash:   "123456",
		Number: uint64(1000),
	})
	b.Add(now.Add(-5*time.Minute), recorder.BlockData{
		Hash:   "654321",
		Number: uint64(1001),
	})
	s := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= 5*time.Minute, "wrong first duration")

	go b.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	s = b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= 10*time.Minute, "wrong second duration")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenOneExpiration(t *testing.T) {
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

	go b.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	s = b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= 2*time.Second, "wrong smaller duration")
	assert.True(t, s.Duration < 2*time.Hour, "wrong larger duration")
}

func TestBlocksRemoveOutdatedPeriodicallyWhenManyExpiration(t *testing.T) {
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().NewTimer(gomock.Any()).Return(time.NewTimer(1)).Times(1)

	now := time.Now()
	b := recorder.NewBlock()
	size := 200
	for i := 0; i < size; i++ {
		b.Add(now.Add(time.Duration(-2*(size-i))*time.Minute), recorder.BlockData{
			Hash:   strconv.Itoa(i),
			Number: uint64(i),
		})
	}
	s := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration >= 2*99*time.Minute, "wrong first duration")

	go b.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	s = b.Summary().(*recorder.BlocksSummary)
	assert.True(t, s.Duration <= 120*time.Minute, "wrong second duration")
}

func TestSummaryWhenForkSingleBlock(t *testing.T) {
	b := recorder.NewBlock()
	now := time.Now()
	blockNumber := uint64(1000)
	b.Add(now, recorder.BlockData{
		Hash:   "1",
		Number: blockNumber,
	})
	b.Add(now, recorder.BlockData{
		Hash:   "2",
		Number: blockNumber,
	})

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, uint64(1), summary.BlockCount, "wrong block count")
	assert.Equal(t, 1, len(summary.Forks), "wrong Fork count")
	assert.Equal(t, blockNumber, summary.Forks[0].Begin, "wrong Fork start")
	assert.Equal(t, blockNumber, summary.Forks[0].End, "wrong Fork end")
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
	assert.Equal(t, uint64(6), summary.BlockCount, "wrong block count")
	assert.Equal(t, 1, len(summary.Forks), "wrong Fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong Fork start")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong Fork end")
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
	assert.Equal(t, uint64(9), summary.BlockCount, "wrong block count")
	assert.Equal(t, 3, len(summary.Forks), "wrong Fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong first Fork begin")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong first Fork begin")
	assert.Equal(t, blockNumber+6, summary.Forks[1].Begin, "wrong second Fork end")
	assert.Equal(t, blockNumber+7, summary.Forks[1].End, "wrong second Fork begin")
	assert.Equal(t, blockNumber+7, summary.Forks[2].Begin, "wrong third Fork begin")
	assert.Equal(t, blockNumber+7, summary.Forks[2].End, "wrong third Fork end")
}

func TestSummaryWhenForkInProgressAndDropThreeBlocks(t *testing.T) {
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
	assert.Equal(t, uint64(6), summary.BlockCount, "wrong block count")
	assert.Equal(t, 1, len(summary.Forks), "wrong Fork count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong Fork begin")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong Fork end")
}

func TestSummaryWhenForkBlockRecycledEntirely(t *testing.T) {
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

	go b.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.True(t, summary.Duration <= time.Second, "wrong duration")
	assert.Equal(t, uint64(3), summary.BlockCount, "wrong block count")
	assert.Equal(t, 1, len(summary.Forks), "wrong Fork count")
	assert.Equal(t, blockNumber+6, summary.Forks[0].Begin, "wrong Fork begin")
	assert.Equal(t, blockNumber+7, summary.Forks[0].End, "wrong Fork end")
}

func TestSummaryWhenForkBlockRecycledPartially(t *testing.T) {
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

	go b.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.Equal(t, 1, len(summary.Forks), "wrong Fork count")
	assert.Equal(t, uint64(3), summary.BlockCount, "wrong block count")
	assert.Equal(t, blockNumber+3, summary.Forks[0].Begin, "wrong Fork begin")
	assert.Equal(t, blockNumber+4, summary.Forks[0].End, "wrong Fork end")
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

	go b.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)

	summary := b.Summary().(*recorder.BlocksSummary)
	assert.Equal(t, 1, len(summary.LongConfirms), "wrong long confirm count")
	assert.Equal(t, blockNumber+2, summary.LongConfirms[0].BlockNumber, "wrong long confirm block number")
}

func TestBlockSummaryValidWhenInvalidOfLongConfirms(t *testing.T) {
	s := recorder.BlocksSummary{
		BlockCount: 10,
		Duration:   time.Hour,
		Forks:      []recorder.Fork{},
		LongConfirms: []recorder.LongConfirm{
			{
				BlockNumber: 1000,
				ExpiredAt:   time.Time{},
				Period:      3 * time.Hour,
				Reported:    false,
			},
		},
	}

	assert.Equal(t, false, s.Valid(), "wrong valid first time")
	assert.Equal(t, true, s.Valid(), "wrong valid second time")
}

func TestBlockSummaryValidWhenInvalidForks(t *testing.T) {
	s := recorder.BlocksSummary{
		BlockCount: 10,
		Duration:   time.Hour,
		Forks: []recorder.Fork{
			{
				Begin:     uint64(1000),
				End:       uint64(1001),
				ExpiredAt: time.Time{},
			},
		},
		LongConfirms: []recorder.LongConfirm{},
	}

	assert.Equal(t, false, s.Valid(), "wrong valid forks")
}

func TestBlockSummaryValidWhenInvalidBlockContinuity(t *testing.T) {
	s := recorder.BlocksSummary{
		BlockCount:    10,
		Duration:      time.Hour,
		Forks:         []recorder.Fork{},
		LongConfirms:  []recorder.LongConfirm{},
		MissingBlocks: []uint64{uint64(1000), uint64(1001)},
	}

	assert.Equal(t, false, s.Valid(), "wrong valid missing blocks")
}

func TestBlockSummaryValidWhenValid(t *testing.T) {
	s := recorder.BlocksSummary{
		BlockCount: 10,
		Duration:   time.Hour,
	}

	assert.Equal(t, true, s.Valid(), "wrong invalid")
}
