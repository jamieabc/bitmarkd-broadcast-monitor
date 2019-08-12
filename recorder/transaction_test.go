package recorder_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

const (
	txID1 = "txID-1"
	txID2 = "txID-2"
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

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestSummaryWhenDropWithDistinct(t *testing.T) {
	setupTransaction()
	r1 := recorder.NewTransaction()
	r2 := recorder.NewTransaction()
	now1 := time.Now()
	now2 := now1.Add(1 * time.Second)
	r1.Add(now1, txID1)
	r2.Add(now2, txID2)

	summary1 := r1.Summary().(*recorder.TransactionSummary)
	summary2 := r2.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, 0.5, summary1.Droprate, "wrong droprate")
	assert.Equal(t, 0.5, summary2.Droprate, "wrong droprate")
}

func TestSummaryWhenDropWithSame(t *testing.T) {
	setupTransaction()
	r1 := recorder.NewTransaction()
	r2 := recorder.NewTransaction()
	now := time.Now()
	r1.Add(now, txID1)
	r2.Add(now, txID1)

	summary1 := r1.Summary().(*recorder.TransactionSummary)
	summary2 := r2.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary1.Droprate, "wrong droprate")
	assert.Equal(t, float64(0), summary2.Droprate, "wrong droprate")
}

func TestTransactionCleanupPeriodicallyWhenExpiration(t *testing.T) {
	setupTransaction()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	now := time.Now()
	r := recorder.NewTransaction()

	txID1 := "txID-1"
	txID2 := "txID-2"
	r.Add(now.Add(-2*expiredTimeInterval), txID1)
	r.Add(now, txID2)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")

	go r.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	transactionShutdownChan <- struct{}{}
	summary = r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, 0.5, summary.Droprate, "wrong droprate")
}

func TestTransactionCleanupPeriodicallyWhenNoExpiration(t *testing.T) {
	setupTransaction()
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	now := time.Now()
	r := recorder.NewTransaction()

	txID1 := "txID-1"
	txID2 := "txID-2"
	r.Add(now.Add(-10*time.Second), txID1)
	r.Add(now, txID2)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")

	go r.RemoveOutdatedPeriodically(mock)
	<-time.After(10 * time.Millisecond)
	transactionShutdownChan <- struct{}{}
	summary = r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}
