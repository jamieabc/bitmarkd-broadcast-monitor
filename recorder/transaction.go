package recorder

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
)

const (
	txArriveDelayTime = 1 * time.Minute
)

//this variable stores superset of all transactions
var globalTransactions *transactions
var shutdownChan <-chan struct{}

type transactions struct {
	sync.Mutex
	data       map[string]expiredAt
	received   bool
	nextItemID int
}

//TransactionSummary - summary of received￿￿￿ transactions
type TransactionSummary struct {
	Droprate      float64
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
	return fmt.Sprintf("earliest received to now got %d transactions, drop percent: %f%%", t.ReceivedCount, dropPercent)
}

//Add - Add transaction
func (t *transactions) Add(receivedTime time.Time, args ...interface{}) {
	if !t.received {
		t.received = true
	}
	t.add(receivedTime, args)
	globalTransactions.add(receivedTime, args)
}

func (t *transactions) add(receivedTime time.Time, args ...interface{}) {
	txID := fmt.Sprintf("%v", args[0])
	if t.isTxIDExist(txID) {
		return
	}

	t.Lock()
	t.data[txID] = expiredAt(receivedTime.Add(txArriveDelayTime))
	t.Unlock()
}

func (t *transactions) isTxIDExist(txID string) bool {
	t.Lock()
	_, ok := t.data[txID]
	t.Unlock()

	return ok
}

//RemoveOutdatedPeriodically - clean expired transaction periodically
func (t *transactions) RemoveOutdatedPeriodically(c clock.Clock) {
	timer := c.After(expiredTimeInterval)
loop:
	for {
		select {
		case <-shutdownChan:
			break loop

		case <-timer:
			cleanupExpiredTransaction(t)
			timer = c.After(txArriveDelayTime)
		}
	}
}

func cleanupExpiredTransaction(t *transactions) {
	now := time.Now()
	t.Lock()
	defer t.Unlock()
	for k, v := range t.data {
		if now.After(time.Time(v)) {
			delete(t.data, k)
		}
	}
}

//Summary - summarize transactions info, mainly droprate
func (t *transactions) Summary() interface{} {
	globalCount := len(globalTransactions.data)
	if 0 == globalCount {
		return &TransactionSummary{
			Droprate:      0,
			received:      t.received,
			ReceivedCount: 0,
		}
	}
	dropRate := float64(globalCount-len(t.data)) / float64(globalCount)
	return &TransactionSummary{
		Droprate:      dropRate,
		received:      t.received,
		ReceivedCount: len(t.data),
	}
}

func initialiseTransactions(shutdown <-chan struct{}) {
	globalTransactions = newTransaction()
	shutdownChan = shutdown
}

func newTransaction() *transactions {
	return &transactions{
		data: make(map[string]expiredAt),
	}
}

//NewTransaction - new transaction
func NewTransaction() Recorder {
	return newTransaction()
}
