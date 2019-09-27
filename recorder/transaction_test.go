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

func TestSummaryWhenEmpty(t *testing.T) {
	r := recorder.NewTransaction()
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestSummaryWhenNoDrop(t *testing.T) {
	r := recorder.NewTransaction()
	now := time.Now()
	r.Add(now, txID1)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestTransactionCleanupPeriodicallyWhenExpiration(t *testing.T) {
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	now := time.Now()
	r := recorder.NewTransaction()

	r.Add(now.Add(-2*expiredTimeInterval), txID1)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(expectedTotalReceivedCount-1)/expectedTotalReceivedCount, summary.Droprate, "wrong droprate")

	go r.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)
	summary = r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(1), summary.Droprate, "wrong droprate")
}

func TestTransactionCleanupPeriodicallyWhenNoExpiration(t *testing.T) {
	ctl, mock := setupTestClock(t)
	defer ctl.Finish()

	mock.EXPECT().After(gomock.Any()).Return(time.After(1)).Times(2)

	now := time.Now()
	nowTruncateToMinute := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	r := recorder.NewTransaction()

	r.Add(nowTruncateToMinute.Add(-2*time.Minute), txID1)
	r.Add(nowTruncateToMinute, txID2)
	r.Add(nowTruncateToMinute.Add(10*time.Second), txID1)
	summary := r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")

	go r.PeriodicRemove([]interface{}{mock, ctx.Done()})
	<-time.After(10 * time.Millisecond)
	summary = r.Summary().(*recorder.TransactionSummary)

	assert.Equal(t, float64(0), summary.Droprate, "wrong droprate")
}

func TestTransactionSummaryValidateWhenInvalid(t *testing.T) {
	s := recorder.TransactionSummary{
		Droprate:      0.2,
		Duration:      time.Minute,
		ReceivedCount: 10,
	}

	assert.Equal(t, false, s.Valid(), "wrong validator")
}
