package records

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
)

const (
	recordSize = 40
)

type Records interface {
	AddHeartbeat(time.Time)
	AddBlock(uint64, blockdigest.Digest)
	BlockSummary() uint64
	HeartbeatSummaryFromTime(time.Time) (time.Duration, uint16, float64)
}

type block struct {
	height uint64
	digest blockdigest.Digest
}

type RecordsImpl struct {
	sync.Mutex
	heartbeats              [recordSize]time.Time
	blocks                  [recordSize]block
	blockIdx                int
	heartbeatIdx            int
	heartbeatIntervalSecond float64
	highestBlock            uint64
}

// New - new records
func New(heartbeatIntervalSecond float64) Records {
	return &RecordsImpl{
		heartbeatIntervalSecond: heartbeatIntervalSecond,
	}
}

// AddHeartbeat - add heartbeat record
func (r *RecordsImpl) AddHeartbeat(t time.Time) {
	r.Lock()
	defer r.Unlock()

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

	r.Lock()
	defer r.Unlock()

	r.blocks[r.blockIdx] = b
	r.blockIdx = nextIdx(r.blockIdx)
	if r.highestBlock < height {
		r.highestBlock = height
	}
}

// BlockSummary - return highest block
func (r *RecordsImpl) BlockSummary() uint64 {
	return r.highestBlock
}

// HeartbeatSummary - heartbeat summary of duration, count, and droprate
func (r *RecordsImpl) HeartbeatSummaryFromTime(durationEndTime time.Time) (time.Duration, uint16, float64) {
	r.Lock()
	defer r.Unlock()

	min := r.heartbeats[0]
	count := uint16(0)

	for i := 0; i < recordSize; i++ {
		if (time.Time{}) != r.heartbeats[i] {
			count++
			if r.heartbeats[i].Before(min) {
				min = r.heartbeats[i]
			}
		} else {
			break
		}
	}

	if (time.Time{}) == min || durationEndTime.Before(min) {
		return time.Duration(0), count, float64(0)
	}

	duration := durationEndTime.Sub(min)

	return duration, count, r.droprate(duration, count)
}

func (r *RecordsImpl) droprate(duration time.Duration, actualReceived uint16) float64 {
	expectedCount := math.Floor(duration.Seconds()/r.heartbeatIntervalSecond) + 1
	fmt.Printf("expectedCount: %f\n", expectedCount)
	fmt.Printf("actual recunt: %d\n", actualReceived)
	if 0 == expectedCount {
		return float64(0)
	}

	return (expectedCount - float64(actualReceived)) * 100 / expectedCount
}
