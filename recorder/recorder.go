package recorder

import (
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
)

const (
	recordSize = 90
)

// Recorder - recorder interface
type Recorder interface {
	Add(time.Time, ...interface{})
	Summary() interface{}
}

type block struct {
	height uint64
	digest blockdigest.Digest
}

type records struct {
	sync.Mutex
	heartbeats              [recordSize]time.Time
	blocks                  [recordSize]block
	blockIdx                int
	heartbeatIdx            int
	heartbeatIntervalSecond float64
	highestBlock            uint64
}

func nextID(idx int) int {
	if recordSize-1 == idx {
		return 0
	} else {
		return idx + 1
	}
}

func prevID(currentID int) int {
	if 0 == currentID {
		return recordSize - 1
	}
	return currentID - 1
}
