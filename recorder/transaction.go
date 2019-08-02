package recorder

import (
	"fmt"
	"sync"
	"time"
)

type transaction struct {
	expiredAt time.Time
	txID      string
}

type transactions struct {
	sync.Mutex
	data       [recordSize]transaction
	nextItemID int
}

//TransactionSummary - summary of received￿￿￿ transactions
type TransactionSummary struct {
	Droprate float64
}

const (
	txArriveDelayTime = 1 * time.Minute
)

// this variable stores superset of each node's transaction
var globalData transactions

//Add - Add transaction
func (t *transactions) Add(receivedTime time.Time, args ...interface{}) {
	arg := args[0]
	txID := fmt.Sprintf("%v", arg)
	if t.isTxIDExist(txID) {
		return
	}

	newTX := transaction{
		expiredAt: receivedTime.Add(txArriveDelayTime),
		txID:      txID,
	}

	t.Lock()
	t.data[t.nextItemID] = newTX
	t.nextItemID = nextID(t.nextItemID)
	t.Unlock()
}

func (t *transactions) isTxIDExist(txID string) bool {
	for _, item := range t.data {
		if (transaction{}) == item {
			return false
		}
		if txID == item.txID {
			return true
		}
	}
	return false
}

//Summary - summarize transaction
func (t *transactions) Summary() interface{} {
	globalCount := recordCount(&globalData)
	if 0 == globalCount {
		return &TransactionSummary{0}
	}
	dropRate := float64(recordCount(t)) / float64(globalCount)
	return &TransactionSummary{dropRate}
}

func recordCount(t *transactions) int {
	if (transaction{}) == t.data[t.nextItemID] {
		return t.nextItemID
	}
	return recordSize
}

//AddTransaction - add transaction
func AddTransaction(t Recorder, receivedTime time.Time, txID string) {
	t.Add(receivedTime, txID)
	globalData.Add(receivedTime, txID)
}

//Initialise
func Initialise() {
	globalData = transactions{}
}

// NewTransaction - new transaction
func NewTransaction() Recorder {
	return &transactions{}
}
