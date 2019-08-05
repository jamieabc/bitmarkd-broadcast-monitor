package recorder_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

var shutdownChan chan struct{}

func init() {
	shutdownChan = make(chan struct{})
}

func setupTransaction() {
	recorder.Initialise(shutdownChan)
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
	txID := "this is test"
	r.Add(now, txID)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestSummaryWhenDropWithDistinct(t *testing.T) {
	setupTransaction()
	r1 := recorder.NewTransaction()
	r2 := recorder.NewTransaction()
	txID1 := "id1"
	txID2 := "id2"
	now1 := time.Now()
	now2 := now1.Add(1 * time.Second)
	recorder.AddTransaction(r1, now1, txID1)
	recorder.AddTransaction(r2, now2, txID2)

	summary1 := r1.Summary().(*recorder.TransactionSummary)
	summary2 := r2.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, 0.5, summary1.Droprate, "wrong droprate")
	assert.Equal(t, 0.5, summary2.Droprate, "wrong droprate")
}

func TestSummaryWhenDropWithSame(t *testing.T) {
	setupTransaction()
	r1 := recorder.NewTransaction()
	r2 := recorder.NewTransaction()
	txID := "id"
	now := time.Now()
	recorder.AddTransaction(r1, now, txID)
	recorder.AddTransaction(r2, now, txID)

	summary1 := r1.Summary().(*recorder.TransactionSummary)
	summary2 := r2.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(1), summary1.Droprate, "wrong droprate")
	assert.Equal(t, float64(1), summary2.Droprate, "wrong droprate")
}
