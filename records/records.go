package records

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
)

const (
	heartbeatRecorderdSize  = 40
	blockRecorderdSize      = 40
	minHeartbeatsForSummary = 10
	minBlocksForSummary     = 10
)

type Records interface {
	AddHeartbeat(time.Time)
	AddBlock(uint64, blockdigest.Digest)
	HighestBlock() uint64
	HeartbeatSummary() (time.Duration, uint16)
}

type block struct {
	number uint64
	digest blockdigest.Digest
}

type RecordsImpl struct {
	heartbeats   [heartbeatRecorderdSize]time.Time
	blocks       [blockRecorderdSize]block
	blockIdx     int
	heartbeatIdx int
}

// Initialise - initialise
func Initialise() Records {
	return &RecordsImpl{
		blockIdx:     0,
		heartbeatIdx: 0,
	}
}

// AddHeartbeat - add heartbeat record
func (r *RecordsImpl) AddHeartbeat(t time.Time) {
	r.heartbeats[r.heartbeatIdx] = t
	r.nextHeartbeatIdx()
}

func (r *RecordsImpl) nextHeartbeatIdx() {
	if r.heartbeatIdx == heartbeatRecorderdSize-1 {
		r.heartbeatIdx = 0
	} else {
		r.heartbeatIdx++
	}
}

// AddBlock - add block record
func (r *RecordsImpl) AddBlock(number uint64, digest blockdigest.Digest) {
	b := block{
		number: number,
		digest: digest,
	}
	r.blocks[r.blockIdx] = b
	r.nextBlockIdx()
}

func (r *RecordsImpl) nextBlockIdx() {
	if r.blockIdx == blockRecorderdSize-1 {
		r.blockIdx = 0
	} else {
		r.blockIdx++
	}
}

// HeartbeatSummary - heartbeat summary of duration and count
func (r *RecordsImpl) HeartbeatSummary() (time.Duration, uint16) {
	max, min := r.heartbeats[0], r.heartbeats[0]
	count := uint16(0)

	for i := 0; i < heartbeatRecorderdSize; i++ {
		if (time.Time{}) != r.heartbeats[i] {
			count++
			if r.heartbeats[i].After(max) {
				max = r.heartbeats[i]
			}
			if r.heartbeats[i].Before(max) {
				min = r.heartbeats[i]
			}
		}
	}

	if (time.Time{}) == max || (time.Time{}) == min {
		return time.Duration(0), count
	}

	return max.Sub(min), count
}

// HighestBlock - highest block number
func (r *RecordsImpl) HighestBlock() uint64 {
	highest := uint64(0)
	for i := 0; i < blockRecorderdSize; i++ {
		if (block{}) != r.blocks[i] && (highest < r.blocks[i].number) {
			highest = r.blocks[i].number
		}
	}
	return highest
}
