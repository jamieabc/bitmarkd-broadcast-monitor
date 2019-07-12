package records_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/records"
	"github.com/stretchr/testify/assert"
)

var (
	defaultDigest = blockdigest.Digest{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
)

func TestInitialise(t *testing.T) {
	r := records.Initialise()

	highestBlock := r.BlockSummary()
	duration, count := r.HeartbeatSummary()

	assert.Equal(t, uint64(0), highestBlock, "wrong initial highest block number")
	assert.Equal(t, time.Duration(0), duration, "wrong initial heartbeat duration")
	assert.Equal(t, uint16(0), count, "wrong initial heartbeat count")
}

func TestBlockSummaryWhenNoCycle(t *testing.T) {
	r := records.Initialise()
	r.AddBlock(uint64(100), defaultDigest)
	r.AddBlock(uint64(101), defaultDigest)
	r.AddBlock(uint64(102), defaultDigest)

	highestBlockNumber := r.BlockSummary()
	assert.Equal(t, uint64(102), highestBlockNumber, "wrong highest block number")
}

func TestBlockSummaryWhenEdge(t *testing.T) {
	r := records.Initialise()
	for i := 0; i < 40; i++ {
		r.AddBlock(uint64(i), defaultDigest)
	}

	highestBlockNumber := r.BlockSummary()
	assert.Equal(t, uint64(39), highestBlockNumber, "wrong highest block number")
}

func TestBlockSummaryWhenCycle(t *testing.T) {
	r := records.Initialise()
	for i := 0; i < 50; i++ {
		r.AddBlock(uint64(i), defaultDigest)
	}

	highestBlockNumber := r.BlockSummary()
	assert.Equal(t, uint64(49), highestBlockNumber, "wrong highest block number")
}

func TestHeartbeatSummaryWhenSingl(t *testing.T) {
	r := records.Initialise()
	r.AddHeartbeat(time.Now())
	duration, count := r.HeartbeatSummary()

	assert.Equal(t, time.Duration(0), duration, "wrong duration")
	assert.Equal(t, uint16(1), count, "wrong heartbeat count")
}

func TestHeartbeatSummaryWhenEnough(t *testing.T) {
	r := records.Initialise()
	now := time.Now()
	for i := 0; i < 15; i++ {
		r.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count := r.HeartbeatSummary()

	assert.Equal(t, time.Duration(14)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(15), count, "wrong heartbeat count")
}

func TestHeartbeatSummaryWhenEdge(t *testing.T) {
	r := records.Initialise()
	now := time.Now()
	for i := 0; i < 40; i++ {
		r.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count := r.HeartbeatSummary()

	assert.Equal(t, time.Duration(39)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(40), count, "wrong heartbeat count")
}

func TestHeartbeatSummaryWhenCycle(t *testing.T) {
	r := records.Initialise()
	now := time.Now()
	for i := 0; i < 50; i++ {
		r.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count := r.HeartbeatSummary()

	assert.Equal(t, time.Duration(39)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(40), count, "wrong heartbeat count")
}
