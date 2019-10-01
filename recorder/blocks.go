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

// Fork - Fork data structure
type Fork struct {
	Begin     uint64
	End       uint64
	ExpiredAt time.Time
}

// LongConfirm - structure to record long confirmation
type LongConfirm struct {
	BlockNumber uint64
	ExpiredAt   time.Time
	Period      time.Duration
	Reported    bool
}

type blocks struct {
	sync.Mutex
	data           [dataLength]BlockData
	earliest       time.Time
	latestBlock    BlockData
	forkInProgress bool
	forks          []Fork
	longConfirms   []LongConfirm
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
	b.forks = append(b.forks, Fork{
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
		b.longConfirms = append(b.longConfirms, LongConfirm{
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
func (b *blocks) PeriodicRemove(args []interface{}) {
	if 2 != len(args) {
		fmt.Printf("blocks PeriodicRemove wrong arguments length\n")
		return
	}
	c := args[0].(clock.Clock)
	shutdown := args[1].(<-chan struct{})
	timer := c.NewTimer(expiredTimeInterval)
loop:
	for {
		select {
		case <-shutdown:
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
	fmt.Println("terminate blocks PeriodicRemove")
}

func cleanupExpiredBlocks(b *blocks, now time.Time) {
	expiredTime := now.Add(-1 * expiredTimeInterval)
	startIdx, endIdx := indexNotFound, indexNotFound
	for i := 0; i < dataLength; i++ {
		if b.data[i].isEmpty() {
			break
		}

		if b.data[i].ReceivedTime.After(expiredTime) {
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
		b.forks = make([]Fork, 0)
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
func (b *blocks) Summary() SummaryOutput {
	duration, blockCount, missingBlocks := summarize(b)
	return &BlocksSummary{
		BlockCount:    blockCount,
		Duration:      duration,
		Forks:         b.forks,
		LongConfirms:  b.longConfirms,
		MissingBlocks: missingBlocks,
	}
}

func summarize(b *blocks) (time.Duration, uint64, []uint64) {
	if (time.Time{}) == b.data[0].ReceivedTime {
		return time.Duration(0), uint64(0), nil
	}

	sorted := sortArray(b)
	count, missing := missingBlocks(sorted)
	return time.Now().Sub(sorted[0].ReceivedTime), uint64(count), missing
}

func sortArray(b *blocks) []BlockData {
	startIndex, endIndex := findBlockStartAndEnd(b)
	if startIndex <= endIndex {
		return b.data[startIndex : endIndex+1]
	}
	return append(b.data[startIndex:dataLength], b.data[:endIndex+1]...)
}

// assumes block number comes in order, blocks may be dropped but never comes
// dis-order, because block is assumed to be generated roughly about 2 minutes,
// thus this assumption should be true
func findBlockStartAndEnd(b *blocks) (startIndex, endIndex int) {
	earliest := b.data[startIndex].Number
	latest := b.data[startIndex].Number

	for i := 1; i < dataLength; i++ {
		if b.data[i].isEmpty() {
			break
		}

		current := b.data[i].Number
		if earliest > current {
			startIndex = i
			earliest = current
		} else if latest < current {
			endIndex = i
			latest = current
		}
	}

	return
}

func missingBlocks(blocks []BlockData) (int, []uint64) {
	mappings := make(map[uint64]string)
	keys := make([]uint64, 0)

	for _, b := range blocks {
		if _, ok := mappings[b.Number]; !ok {
			mappings[b.Number] = ""
			keys = append(keys, b.Number)
		}
	}

	missing := make([]uint64, 0)
	for i := blocks[0].Number; i < blocks[len(blocks)-1].Number; i++ {
		if _, ok := mappings[i]; !ok {
			missing = append(missing, i)
		}
	}
	return len(keys), missing
}

// BlocksSummary - BlockData summary data structure
type BlocksSummary struct {
	BlockCount    uint64
	Duration      time.Duration
	Forks         []Fork
	LongConfirms  []LongConfirm // confirm time longer than 30 minutes
	MissingBlocks []uint64
}

func (b *BlocksSummary) String() string {
	return fmt.Sprintf(
		"receive %d blolcks in %s, forks: %d, long confirms: %d, missing blocks: %v\n%s\n%s",
		b.BlockCount,
		b.Duration,
		len(b.Forks),
		len(b.LongConfirms),
		b.MissingBlocks,
		forkInfo(b.Forks),
		confirmInfo(b.LongConfirms),
	)
}

// Valid - any long confirmation, fork, or block not continuous that has not reported
func (b *BlocksSummary) Valid() bool {
	var reported []LongConfirm
	if 0 == len(b.LongConfirms) && 0 == len(b.Forks) && 0 == len(b.MissingBlocks) {
		return true
	}

	if 0 < len(b.MissingBlocks) {
		return false
	}

	for _, c := range b.LongConfirms {
		if !c.Reported {
			c.Reported = true
			reported = append(reported, c)
		}
	}

	if 0 < len(reported) || 0 < len(b.Forks) {
		b.LongConfirms = reported
		return false
	}

	return true
}

func forkInfo(forks []Fork) string {
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

func confirmInfo(longConfirms []LongConfirm) string {
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
		earliest:     time.Now(),
		forks:        make([]Fork, 0),
		longConfirms: make([]LongConfirm, 0),
	}
}
