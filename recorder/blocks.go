package recorder

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

const (
	dataLength          = 100
	longConfirmInterval = 30 * time.Minute
)

// BlockData - data of a block
type BlockData struct {
	Hash         string
	Number       uint64
	GenerateTime time.Time
	ReceivedTime time.Time
}

func (b BlockData) isEmpty() bool {
	return uint64(0) == b.Number || "" == b.Hash
}

// fork - fork data structure
type fork struct {
	Begin     uint64
	End       uint64
	ExpiredAt time.Time
}

type longConfirm struct {
	BlockNumber uint64
	ExpiredAt   time.Time
	Period      time.Duration
}

type blocks struct {
	sync.Mutex
	data           [dataLength]BlockData
	latestBlock    BlockData
	forkInProgress bool
	forks          []fork
	longConfirms   []longConfirm
	id             int
}

// Add - add BlockData item
func (b *blocks) Add(t time.Time, args ...interface{}) {
	nextBlock := args[0].(BlockData)
	nextBlock.ReceivedTime = t
	processFork(b, nextBlock)
	b.addBlock(nextBlock)
	b.updateLatestBlock(nextBlock)
	b.updateLongConfirmPeriodInfo(nextBlock)
}

func processFork(b *blocks, nextBlock BlockData) {
	if isFork(b, nextBlock) {
		updateForkInfo(b, nextBlock)
	} else {
		resetForkStatus(b)
	}
}

// each BlockData should only be broadcast once
func isFork(b *blocks, nextBlock BlockData) bool {
	if b.latestBlock.isEmpty() {
		return false
	}

	if b.latestBlock.Number > nextBlock.Number {
		return true
	}

	if b.latestBlock.Number == nextBlock.Number && b.latestBlock.Hash != nextBlock.Hash {
		return true
	}

	return false
}

func updateForkInfo(b *blocks, nextBlock BlockData) {
	if !b.forkInProgress {
		processFirstForkBlock(b, nextBlock)
	} else {
		if isForkFinish(b, nextBlock) {
			b.forkInProgress = false
		}
	}
}

func processFirstForkBlock(b *blocks, nextBlock BlockData) {
	if b.latestBlock.Number != nextBlock.Number {
		b.forkInProgress = true
	}
	b.forks = append(b.forks, fork{
		Begin:     nextBlock.Number,
		End:       b.latestBlock.Number,
		ExpiredAt: nextBlock.ReceivedTime.Add(expiredTimeInterval),
	})
}

func isForkFinish(b *blocks, nextBlock BlockData) bool {
	return b.latestBlock.Number == nextBlock.Number
}

// in case block broadcast is dropped, make sure when higher BlockData comes, reset flag
func resetForkStatus(b *blocks) {
	if b.forkInProgress {
		b.forkInProgress = false
	}
}

func (b *blocks) addBlock(nextBlock BlockData) {
	b.data[b.id] = nextBlock
	nextID(b)
}

func (b *blocks) updateLatestBlock(nextBlock BlockData) {
	if nextBlock.Number >= b.latestBlock.Number && nextBlock.Hash != b.latestBlock.Hash {
		b.latestBlock = nextBlock
	}
}

func (b *blocks) updateLongConfirmPeriodInfo(nextBlock BlockData) {
	confirmInterval := nextBlock.ReceivedTime.Sub(nextBlock.GenerateTime)
	if confirmInterval >= longConfirmInterval {
		b.longConfirms = append(b.longConfirms, longConfirm{
			BlockNumber: nextBlock.Number,
			ExpiredAt:   nextBlock.ReceivedTime.Add(expiredTimeInterval),
			Period:      confirmInterval,
		})
	}
}

func nextID(b *blocks) {
	if dataLength-1 == b.id {
		b.id = 0
	} else {
		b.id++
	}
}

// PeriodicRemove - periodically remove outdated item
func (b *blocks) PeriodicRemove(c clock.Clock) {
	timer := c.NewTimer(expiredTimeInterval)
loop:
	for {
		select {
		case <-shutdownChan:
			break loop

		case <-timer.C:
			now := time.Now()
			b.Lock()
			cleanupExpiredBlocks(b, now)
			cleanupExpiredForks(b, now)
			cleanupExpiredLongConfirms(b, now)
			b.Unlock()
			timer.Reset(expiredTimeInterval)
		}
	}
}

func cleanupExpiredBlocks(b *blocks, now time.Time) {
	expiredTime := now.Add(-1 * expiredTimeInterval)
	startIdx, endIdx := indexNotFound, indexNotFound
	for i := 0; i < dataLength; i++ {
		if !b.data[i].isEmpty() && b.data[i].ReceivedTime.After(expiredTime) {
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

	var tempArray [dataLength]BlockData
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

func cleanupExpiredLongConfirms(b *blocks, now time.Time) {
	var startIdx int
	for i, c := range b.longConfirms {
		if now.Before(c.ExpiredAt) {
			startIdx = i
		}
	}
	if 0 != startIdx {
		b.longConfirms = b.longConfirms[startIdx:]
	}
}

// Summary - summarize blocks stat
func (b *blocks) Summary() interface{} {
	duration, startBlock, endBlock := summarize(b)
	return &BlocksSummary{
		BlockCount:   countBlock(startBlock, endBlock),
		Duration:     duration,
		Forks:        b.forks,
		LongConfirms: b.longConfirms,
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
	if (time.Time{}) == b.data[0].ReceivedTime {
		return time.Duration(0), uint64(0), uint64(0)
	}

	earliestTime := b.data[0].ReceivedTime
	var index int
	for i := 0; i < dataLength; i++ {
		t := b.data[i].ReceivedTime
		if (time.Time{}) == t {
			break
		}
		if earliestTime.After(t) {
			earliestTime = t
			index = i
			continue
		}
	}

	// in case array is not fully filled, need to check previous BlockData
	prevBlockNumber := prevBlockNumber(b.data, index)
	nextBlockNumber := nextBlockNumber(b.data, index)
	var endBlockNumber uint64
	if prevBlockNumber >= nextBlockNumber {
		endBlockNumber = prevBlockNumber
	} else {
		endBlockNumber = nextBlockNumber
	}
	return time.Now().Sub(earliestTime), b.data[index].Number, endBlockNumber
}

func nextBlockNumber(b [dataLength]BlockData, index int) uint64 {
	nextBlockID := index + 1
	if dataLength-1 == index {
		nextBlockID = 0
	}
	return b[nextBlockID].Number
}

func prevBlockNumber(b [dataLength]BlockData, index int) uint64 {
	prevBlockID := index - 1
	if 0 == index {
		prevBlockID = dataLength - 1
	}
	return b[prevBlockID].Number
}

// BlocksSummary - BlockData summary data structure
type BlocksSummary struct {
	BlockCount   uint64
	Duration     time.Duration
	Forks        []fork
	LongConfirms []longConfirm // confirm time longer than 30 minutes
}

func (b *BlocksSummary) String() string {
	return fmt.Sprintf(
		"receive %d blolcks in %s, forks: %d, long confirms: %d\n%s\n%s",
		b.BlockCount,
		b.Duration,
		len(b.Forks),
		len(b.LongConfirms),
		forkInfo(b.Forks),
		confirmInfo(b.LongConfirms),
	)
}

func forkInfo(forks []fork) string {
	if 0 < len(forks) {
		var str strings.Builder
		str.WriteString("forks info: ")
		for _, f := range forks {
			str.WriteString(fmt.Sprintf("%d to %d ", f.Begin, f.End))
		}
		return str.String()
	}
	return ""
}

func confirmInfo(longConfirms []longConfirm) string {
	if 0 < len(longConfirms) {
		var str strings.Builder
		str.WriteString("long confirms: ")
		for _, c := range longConfirms {
			str.WriteString(fmt.Sprintf("block %d takes %d ", c.BlockNumber, c.Period))
		}
		return str.String()
	}
	return ""
}

// NewBlock - new blocks data structure
func NewBlock() Recorder {
	return &blocks{
		forks:        make([]fork, 0),
		longConfirms: make([]longConfirm, 0),
	}
}
