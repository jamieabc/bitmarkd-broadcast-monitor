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

// fork - fork data structure
type fork struct {
	Begin     uint64
	End       uint64
	ExpiredAt time.Time
}

type blocks struct {
	sync.Mutex
	data           [dataLength]block
	latestBlock    block
	forkInProgress bool
	forks          []fork
	id             int
}

// Add - add block item
func (b *blocks) Add(t time.Time, args ...interface{}) {
	nextBlock := newBlock(t, args)
	processFork(b, nextBlock)
	b.addBlock(nextBlock)
	b.updateLatestBlock(nextBlock)
}

func newBlock(t time.Time, args []interface{}) block {
	number, hash := parseBlockArguments(args)
	return block{
		hash:         hash,
		number:       number,
		receivedTime: t,
	}
}

func parseBlockArguments(args []interface{}) (uint64, string) {
	number := args[0].(uint64)
	hash := args[1].(string)
	return number, hash
}

func processFork(b *blocks, nextBlock block) {
	if isFork(b, nextBlock) {
		updateForkInfo(b, nextBlock)
	} else {
		resetForkFlag(b)
	}
}

// each block should only be broadcast once
func isFork(b *blocks, nextBlock block) bool {
	if b.latestBlock.isEmpty() {
		return false
	}

	if b.latestBlock.number > nextBlock.number {
		return true
	}

	if b.latestBlock.number == nextBlock.number && b.latestBlock.hash != nextBlock.hash {
		return true
	}

	return false
}

func updateForkInfo(b *blocks, nextBlock block) {
	if isForkNotStart(b) {
		processForkStart(b, nextBlock)
	} else {
		if isForkLastBlock(b, nextBlock) {
			b.forkInProgress = false
		}
	}
}

func isForkNotStart(b *blocks) bool {
	return !b.forkInProgress
}

func processForkStart(b *blocks, nextBlock block) {
	if b.latestBlock.number != nextBlock.number {
		b.forkInProgress = true
	}
	b.forks = append(b.forks, fork{
		Begin:     nextBlock.number,
		End:       b.latestBlock.number,
		ExpiredAt: nextBlock.receivedTime.Add(expiredTimeInterval),
	})
}

func isForkLastBlock(b *blocks, nextBlock block) bool {
	return b.latestBlock.number == nextBlock.number
}

// in case block broadcast is dropped, make sure when higher block comes, reset flag
func resetForkFlag(b *blocks) {
	if b.forkInProgress {
		b.forkInProgress = false
	}
}

func (b *blocks) addBlock(nextBlock block) {
	b.data[b.id] = nextBlock
	nextID(b)
}

func (b *blocks) updateLatestBlock(nextBlock block) {
	if nextBlock.number >= b.latestBlock.number && nextBlock.hash != b.latestBlock.hash {
		b.latestBlock = nextBlock
	}
}

func nextID(b *blocks) {
	if dataLength-1 == b.id {
		b.id = 0
	} else {
		b.id++
	}
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
			now := time.Now()
			cleanupExpiredBlocks(b, now)
			cleanupExpiredForks(b, now)
			timer.Reset(expiredTimeInterval)
		}
	}
}

func cleanupExpiredBlocks(b *blocks, now time.Time) {
	expiredTime := now.Add(-1 * expiredTimeInterval)
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

func cleanupExpiredForks(b *blocks, now time.Time) {
	endIdx := indexNotFound
	for i, f := range b.forks {
		if now.After(f.ExpiredAt) {
			endIdx = i
		}
	}

	if endIdx >= len(b.forks) {
		b.forks = make([]fork, 0)
	} else {
		b.forks = b.forks[endIdx+1:]
	}
}

// Summary - summarize blocks stat
func (b *blocks) Summary() interface{} {
	duration, startBlock, endBlock := summarize(b)
	return &BlocksSummary{
		BlockCount: countBlock(startBlock, endBlock),
		Duration:   duration,
		Forks:      b.forks,
	}
}

func countBlock(startBlock uint64, endBlock uint64) uint64 {
	if 0 == startBlock && 0 == endBlock {
		return uint64(0)
	} else if 0 == endBlock {
		return uint64(1)
	} else {
		return endBlock - startBlock + 1
	}
}

func summarize(b *blocks) (time.Duration, uint64, uint64) {
	if (time.Time{}) == b.data[0].receivedTime {
		return time.Duration(0), uint64(0), uint64(0)
	}

	earliestTime := b.data[0].receivedTime
	var index int
	for i := 0; i < dataLength; i++ {
		t := b.data[i].receivedTime
		if (time.Time{}) == t {
			break
		}
		if earliestTime.After(t) {
			earliestTime = t
			index = i
			continue
		}
	}

	var endBlock uint64
	if 0 == index {
		endBlock = b.data[dataLength-1].number
	} else {
		endBlock = b.data[index-1].number
	}
	return time.Now().Sub(earliestTime), b.data[index].number, endBlock
}

// BlocksSummary - block summary data structure
type BlocksSummary struct {
	BlockCount uint64
	Duration   time.Duration
	Forks      []fork
}

func (b *BlocksSummary) String() string {
	return fmt.Sprintf("receive %d blolcks in %s, forks: %d", b.BlockCount, b.Duration, len(b.Forks))
}

// NewBlock - new blocks data structure
func NewBlock() Recorder {
	return &blocks{
		forks: make([]fork, 0),
	}
}
