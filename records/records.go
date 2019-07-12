package records

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
)

const (
	recordSize = 40
)

type Records interface {
	AddHeartbeat(time.Time)
	AddBlock(uint64, blockdigest.Digest)
	HighestBlock() uint64
	HeartbeatSummary() (time.Duration, uint16)
}

type block struct {
	height uint64
	digest blockdigest.Digest
}

type RecordsImpl struct {
	heartbeats   [recordSize]time.Time
	blocks       [recordSize]block
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
	r.heartbeatIdx = nextIdx(r.heartbeatIdx)
}

func nextIdx(idx int) int {
	if recordSize-1 == idx {
		return 0
	} else {
		return idx + 1
	}
}

// AddBlock - add block record
func (r *RecordsImpl) AddBlock(height uint64, digest blockdigest.Digest) {
	b := block{
		height: height,
		digest: digest,
	}
	r.blocks[r.blockIdx] = b
	r.blockIdx = nextIdx(r.blockIdx)
}

// HeartbeatSummary - heartbeat summary of duration and count
func (r *RecordsImpl) HeartbeatSummary() (time.Duration, uint16) {
	max := r.heartbeats[0]
	maxIdx := 0
	count := uint16(0)

	for i := 0; i < recordSize; i++ {
		if (time.Time{}) != r.heartbeats[i] {
			count++
			if r.heartbeats[i].After(max) {
				max = r.heartbeats[i]
				maxIdx = i
			}
		} else {
			break
		}
	}

	min := r.minHeartbeatTimeAt(maxIdx)

	if (time.Time{}) == max || (time.Time{}) == min {
		return time.Duration(0), count
	}

	return max.Sub(min), count
}

func (r *RecordsImpl) minHeartbeatTimeAt(idx int) time.Time {
	next := nextIdx(idx)
	if (time.Time{}) != r.heartbeats[next] {
		return r.heartbeats[next]
	}
	return r.heartbeats[0]
}

// HighestBlock - highest block height
func (r *RecordsImpl) HighestBlock() uint64 {
	highest := uint64(0)
	for i := 0; i < recordSize; i++ {
		if (block{}) == r.blocks[i] {
			break
		}

		if highest <= r.blocks[i].height {
			highest = r.blocks[i].height
		} else {
			break
		}
	}
	return highest
}
