package recorder

import (
	"fmt"
	"sync"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

const (
	dataLength = 100
)

type block struct {
	hash         string
	number       uint64
	receivedTime time.Time
}

func (b block) isEmpty() bool {
	return uint64(0) == b.number || "" == b.hash
}

type blocks struct {
	sync.Mutex
	data [dataLength]block
	id   int
}

// Add - add block item
func (b *blocks) Add(t time.Time, args ...interface{}) {
	number := args[0].(uint64)
	hash := args[1].(string)
	b.data[b.id] = block{
		hash:         hash,
		number:       number,
		receivedTime: t,
	}
	nextID(b)
}

func nextID(b *blocks) {
	if dataLength-1 == b.id {
		b.id = 0
	}
	b.id++
}

// RemoveOutdatedPeriodically - remove outdated item
func (b *blocks) RemoveOutdatedPeriodically(c clock.Clock) {
	timer := c.NewTimer(expiredTimeInterval)
loop:
	for {
		select {
		case <-shutdownChan:
			break loop

		case <-timer.C:
			cleanupExpiredBlocks(b)
			timer.Reset(expiredTimeInterval)
		}
	}
}

func cleanupExpiredBlocks(b *blocks) {
	expiredTime := time.Now().Add(-1 * expiredTimeInterval)
	startIdx, endIdx := indexNotFound, indexNotFound
	for i := 0; i < dataLength; i++ {
		if !b.data[i].isEmpty() && b.data[i].receivedTime.After(expiredTime) {
			if indexNotFound == startIdx {
				startIdx = i
				endIdx = i
			} else {
				endIdx = i
			}
		}
	}

	if indexNotFound == startIdx {
		return
	}

	var tempArray [dataLength]block
	copy(tempArray[:], b.data[startIdx:(endIdx+1)])
	b.data = tempArray
	b.id = endIdx - startIdx + 1
}

// Summary - summarize blocks stat
func (b *blocks) Summary() interface{} {
	return BlocksSummary{
		Duration: findDuration(b),
	}
}

func findDuration(b *blocks) time.Duration {
	if (time.Time{}) == b.data[0].receivedTime {
		return time.Duration(0)
	}

	earliestTime := b.data[0].receivedTime
	for i := 0; i < dataLength; i++ {
		t := b.data[i].receivedTime
		if (time.Time{}) == t {
			break
		}
		if earliestTime.After(t) {
			earliestTime = t
			continue
		}
	}
	return time.Now().Sub(earliestTime)
}

// Fork - fork data structure
type Fork struct {
	Begin uint64
	End   uint64
}

// BlocksSummary - block summary data structure
type BlocksSummary struct {
	Duration time.Duration
	Forks    []Fork
}

func (b *BlocksSummary) String() string {
	return fmt.Sprintf("duration %s, forks: %d", b.Duration, len(b.Forks))
}

// NewBlocks - new blocks data structure
func NewBlocks() Recorder {
	return &blocks{}
}
