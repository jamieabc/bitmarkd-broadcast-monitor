package recorder

import (
	"fmt"
	"sync"
	"time"
)

type expiredAt time.Time

type transactions struct {
	sync.Mutex
	data       map[string]expiredAt
	nextItemID int
}

//TransactionSummary - summary of received￿￿￿ transactions
type TransactionSummary struct {
	Droprate float64
}

const (
	txArriveDelayTime = 1 * time.Minute
)

// this variable stores superset of all transactions
var globalData transactions

//Add - Add transactions
func (t *transactions) Add(receivedTime time.Time, args ...interface{}) {
	arg := args[0]
	txID := fmt.Sprintf("%v", arg)
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

//Summary - summarize transactions info, mainly droprate
func (t *transactions) Summary() interface{} {
	globalCount := recordCount(&globalData)
	if 0 == globalCount {
		return &TransactionSummary{0}
	}
	dropRate := float64(recordCount(t)) / float64(globalCount)
	return &TransactionSummary{dropRate}
}

func recordCount(t *transactions) int {
	return len(t.data)
}

//AddTransaction - add transation
func AddTransaction(t Recorder, receivedTime time.Time, txID string) {
	t.Add(receivedTime, txID)
	globalData.Add(receivedTime, txID)
}

//Initialise
func Initialise() {
	globalData = transactions{
		data: make(map[string]expiredAt),
	}
}

// NewTransaction - new transaction
func NewTransaction() Recorder {
	return &transactions{
		data: make(map[string]expiredAt),
	}
}
