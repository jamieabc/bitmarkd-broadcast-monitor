package recorder

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

// this variable stores superset of all transactions
var shutdownChan <-chan struct{}

type transactions struct {
	sync.Mutex
	data                  [totalReceivedCount]bool
	firstItemReceivedTime time.Time
	received              bool
}

// TransactionSummary - summary of received￿￿￿ transactions
type TransactionSummary struct {
	Droprate      float64
	Duration      time.Duration
	received      bool
	ReceivedCount int
}

func (t *TransactionSummary) String() string {
	if !t.received {
		return "not receive any transaction yet"
	}

	if t.received && 0 == t.ReceivedCount {
		return "not receive transaction for more than 2 hours"
	}

	dropPercent := math.Floor(t.Droprate*10000) / 100
	return fmt.Sprintf("earliest received to now %s, got %d transactions, drop percent: %f%%", t.Duration, t.ReceivedCount, dropPercent)
}

// Validate - determine if this summary needs to notify
func (t *TransactionSummary) Validate() bool {
	return t.Droprate <= 0.1
}

// Add - Add transaction
func (t *transactions) Add(receivedTime time.Time, args ...interface{}) {
	if !t.received {
		t.received = true
		t.firstItemReceivedTime = roundTimeToMinute(receivedTime)
	}
	t.add(receivedTime, args)
}

func roundTimeToMinute(source time.Time) time.Time {
	return time.Date(source.Year(), source.Month(), source.Day(), source.Hour(), source.Minute(), 0, 0, source.Location())
}

func (t *transactions) add(receivedTime time.Time, args ...interface{}) {
	timeDiff := roundTimeToMinute(receivedTime).Sub(t.firstItemReceivedTime)
	index := int(timeDiff / time.Minute)
	if 0 > index || totalReceivedCount <= index {
		return
	}

	t.data[index] = true
}

// PeriodicRemove - clean expired transaction periodically
func (t *transactions) PeriodicRemove(c clock.Clock) {
loop:
	for {
		select {
		case <-shutdownChan:
			break loop

		case <-c.After(expiredTimeInterval):
			cleanupExpiredTransaction(t)
		}
	}
}

func cleanupExpiredTransaction(t *transactions) {
	if t.firstItemReceivedTime.After(time.Now().Add(-1 * expiredTimeInterval)) {
		return
	}

	t.Lock()
	defer t.Unlock()

	firstItem := findNextReceivedItemIndex(t)

	var tmpArray [totalReceivedCount]bool

	if indexNotFound == firstItem {
		t.firstItemReceivedTime = time.Time{}
	} else {
		copy(tmpArray[firstItem:], t.data[firstItem:])
		t.firstItemReceivedTime = t.firstItemReceivedTime.Add(time.Duration(firstItem+1) * time.Minute)
	}
	t.data = tmpArray
}

// findNextReceivedItemIndex - find first item in the array its value is true
// if nothing found, return indexNotFound(-1)
func findNextReceivedItemIndex(t *transactions) int {
	return findReceivedItemFromIndex(t, 1)
}

func findReceivedItemFromIndex(t *transactions, start int) int {
	for i := start; i < len(t.data); i++ {
		if t.data[i] {
			return i
		}
	}
	return indexNotFound
}

// Summary - summarize transactions info, mainly droprate
func (t *transactions) Summary() SummaryOutput {
	if indexNotFound == findFirstReceivedItemIndex(t) {
		droprate := float64(0)
		if t.received {
			droprate = float64(1)
		}
		return &TransactionSummary{
			Droprate: droprate,
			received: t.received,
		}
	}

	expectedCount := expectedCount(t)
	receivedCount := receivedCountFromPrevTwoHour(t.data)
	dropRate := float64(0)
	if expectedCount > receivedCount {
		dropRate = (expectedCount - receivedCount) / expectedCount
	}
	return &TransactionSummary{
		Droprate:      dropRate,
		Duration:      time.Now().Sub(t.firstItemReceivedTime),
		received:      t.received,
		ReceivedCount: int(receivedCountFromPrevTwoHour(t.data)),
	}
}

func expectedCount(t *transactions) float64 {
	expectedCount := float64(time.Now().Sub(t.firstItemReceivedTime) / time.Minute)
	if 0 == expectedCount {
		return 1
	}
	if float64(totalReceivedCount) < expectedCount {
		return float64(totalReceivedCount)
	}
	return expectedCount
}

func findFirstReceivedItemIndex(t *transactions) int {
	return findReceivedItemFromIndex(t, 0)
}

func receivedCountFromPrevTwoHour(data [totalReceivedCount]bool) float64 {
	count := float64(0)
	for _, value := range data {
		if value {
			count++
		}
	}
	return count
}

func initialiseTransactions(shutdown <-chan struct{}) {
	shutdownChan = shutdown
}

// NewTransaction - new transaction
func NewTransaction() Recorder {
	return &transactions{}
}
