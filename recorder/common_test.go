package recorder_test

import (
	"testing"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"

	"github.com/golang/mock/gomock"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder/mocks"
)

const (
	expiredTimeInterval = 2 * time.Hour
)

var shutdownChan chan struct{}

func init() {
	shutdownChan = make(chan struct{})
}

func setupTestClock(t *testing.T) (*gomock.Controller, *mocks.MockClock) {
	ctl := gomock.NewController(t)
	return ctl, mocks.NewMockClock(ctl)
}

func setupRecorder() {
	recorder.Initialise(shutdownChan)
}
