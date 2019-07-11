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
	records := records.Initialise()

	highestBlock := records.HighestBlock()
	duration, count := records.HeartbeatSummary()

	assert.Equal(t, uint64(0), highestBlock, "wrong initial highest block number")
	assert.Equal(t, time.Duration(0), duration, "wrong initial heartbeat duration")
	assert.Equal(t, uint16(0), count, "wrong initial heartbeat count")
}

func TestHighestBlockWhenNoCycle(t *testing.T) {
	records := records.Initialise()
	records.AddBlock(uint64(100), defaultDigest)
	records.AddBlock(uint64(101), defaultDigest)
	records.AddBlock(uint64(102), defaultDigest)

	highestBlockNumber := records.HighestBlock()
	assert.Equal(t, uint64(102), highestBlockNumber, "wrong highest block number")
}

func TestHighestBlockWhenCycle(t *testing.T) {
	records := records.Initialise()
	for i := 0; i < 50; i++ {
		records.AddBlock(uint64(i), defaultDigest)
	}

	highestBlockNumber := records.HighestBlock()
	assert.Equal(t, uint64(49), highestBlockNumber, "wrong highest block number")
}

func TestHeartbeatSummaryWhenNotEnough(t *testing.T) {
	records := records.Initialise()
	records.AddHeartbeat(time.Now())
	duration, count := records.HeartbeatSummary()

	assert.Equal(t, time.Duration(0), duration, "wrong duration")
	assert.Equal(t, uint16(1), count, "wrong heartbeat count")
}

func TestHeartbeatSummaryWhenEnough(t *testing.T) {
	records := records.Initialise()
	now := time.Now()
	for i := 0; i < 15; i++ {
		records.AddHeartbeat(now.Add(time.Duration(i) * time.Second))
	}
	duration, count := records.HeartbeatSummary()

	assert.Equal(t, time.Duration(14)*time.Second, duration, "wrong duration")
	assert.Equal(t, uint16(15), count, "wrong heartbeat count")
}
