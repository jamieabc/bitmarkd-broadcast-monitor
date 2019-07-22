package records_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/records"
	"github.com/stretchr/testify/assert"
)

const (
	heartbeatInterval = 1
)

var (
	defaultDigest = blockdigest.Digest{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
)

func TestNew(t *testing.T) {
	r := records.New(heartbeatInterval)

	highestBlock := r.BlockSummary()
	duration, count, droprate := r.HeartbeatSummaryFromTime(time.Now())

	assert.Equal(t, uint64(0), highestBlock, "wrong initial highest block number")
	assert.Equal(t, time.Duration(0), duration, "wrong initial heartbeat duration")
	assert.Equal(t, uint16(0), count, "wrong initial heartbeat count")
	assert.Equal(t, float64(0), droprate, "wrong initial drop rate")
}

func TestBlockSummaryWhenNoCycle(t *testing.T) {
	r := records.New(heartbeatInterval)
	r.AddBlock(uint64(100), defaultDigest)
	r.AddBlock(uint64(101), defaultDigest)
	r.AddBlock(uint64(102), defaultDigest)

	highestBlockNumber := r.BlockSummary()
	assert.Equal(t, uint64(102), highestBlockNumber, "wrong highest block number")
}

func TestBlockSummaryWhenEdge(t *testing.T) {
	r := records.New(heartbeatInterval)
	for i := 0; i < 40; i++ {
		r.AddBlock(uint64(i), defaultDigest)
	}

	highestBlockNumber := r.BlockSummary()
	assert.Equal(t, uint64(39), highestBlockNumber, "wrong highest block number")
}

func TestBlockSummaryWhenCycle(t *testing.T) {
	r := records.New(heartbeatInterval)
	for i := 0; i < 50; i++ {
		r.AddBlock(uint64(i), defaultDigest)
	}

	highestBlockNumber := r.BlockSummary()
	assert.Equal(t, uint64(49), highestBlockNumber, "wrong highest block number")
}

func TestHeartbeatSummaryWhenSingle(t *testing.T) {
	r := records.New(heartbeatInterval)
	now := time.Now()
	r.AddHeartbeat(now)
	duration, count, droprate := r.HeartbeatSummaryFromTime(now)

	assert.Equal(t, time.Duration(0), duration, "wrong duration")
	assert.Equal(t, uint16(1), count, "wrong heartbeat count")
	assert.Equal(t, float64(0), droprate, "wrong drop rate")
}

func TestHeartbeatSummaryWhenEnough(t *testing.T) {
	r := records.New(heartbeatInterval)
	now := time.Now()
	size := 15
	for i := 0; i < size; i++ {
		r.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count, droprate := r.HeartbeatSummaryFromTime(now.Add(time.Duration(size-1) * time.Second))

	assert.Equal(t, time.Duration(14)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(15), count, "wrong heartbeat count")
	assert.Equal(t, float64(0), droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenEdge(t *testing.T) {
	r := records.New(heartbeatInterval)
	now := time.Now()
	size := 40
	for i := 0; i < size; i++ {
		r.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count, droprate := r.HeartbeatSummaryFromTime(now.Add(time.Duration(size-1) * time.Second))

	assert.Equal(t, time.Duration(39)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(40), count, "wrong heartbeat count")
	assert.Equal(t, float64(0), droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenCycle(t *testing.T) {
	r := records.New(heartbeatInterval)
	now := time.Now()
	size := 50
	for i := 0; i < size; i++ {
		r.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count, droprate := r.HeartbeatSummaryFromTime(now.Add(time.Duration(size-1) * time.Second))

	assert.Equal(t, time.Duration(39)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(40), count, "wrong heartbeat count")
	assert.Equal(t, float64(0), droprate, "wrong droprate")
}

func TestHeartbeatSummaryWhenDrop(t *testing.T) {
	r := records.New(heartbeatInterval)
	now := time.Now()
	size := 20
	for i := 0; i < size; i++ {
		if 0 < i && 0 == i%5 {
			continue
		}
		r.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count, droprate := r.HeartbeatSummaryFromTime(now.Add(time.Duration(size-1) * time.Second))

	assert.Equal(t, time.Duration(size-1)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(size-3), count, "wrong heartbeat count")
	assert.Equal(t, float64(3*100/size), droprate, "wrong droprate")
}
