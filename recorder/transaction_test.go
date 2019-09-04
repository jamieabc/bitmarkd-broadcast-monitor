package recorder_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

const (
	txID1                      = "txID-1"
	txID2                      = "txID-2"
	expectedTotalReceivedCount = 120
)

var transactionShutdownChan chan struct{}

func init() {
	transactionShutdownChan = make(chan struct{})
}

func setupTransaction() {
	recorder.Initialise(transactionShutdownChan)
}

func TestSummaryWhenEmpty(t *testing.T) {
	setupTransaction()
	r := recorder.NewTransaction()
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestSummaryWhenNoDrop(t *testing.T) {
	setupTransaction()
	r := recorder.NewTransaction()
	now := time.Now()
	r.Add(now, txID1)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(120-1)/120, summary.Droprate, "wrong droprate")
}

func TestTransactionCleanupPeriodicallyWhenExpiration(t *testing.T) {
	setupTransaction()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	now := time.Now()
	r := recorder.NewTransaction()

	r.Add(now.Add(-2*expiredTimeInterval), txID1)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(expectedTotalReceivedCount-1)/expectedTotalReceivedCount, summary.Droprate, "wrong droprate")

	go r.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	//transactionShutdownChan <- struct{}{}
	summary = r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(1), summary.Droprate, "wrong droprate")
}

func TestTransactionCleanupPeriodicallyWhenNoExpiration(t *testing.T) {
	setupTransaction()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	now := time.Now()
	r := recorder.NewTransaction()

	r.Add(now.Add(-2*time.Minute), txID1)
	r.Add(now, txID2)
	r.Add(now.Add(10*time.Second), txID1)
	summary := r.Summary().(*recorder.TransactionSummary)
	droprate := float64(expectedTotalReceivedCount-2) / expectedTotalReceivedCount

	assert.Equal(t, droprate, summary.Droprate, "wrong droprate")

	go r.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	transactionShutdownChan <- struct{}{}
	summary = r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, droprate, summary.Droprate, "wrong droprate")
}
