package db_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bitmark-inc/logger"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db/mocks"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
)

const (
	originalIntervalSecond = 5 * time.Second
	looperIntervalSecond   = originalIntervalSecond + 1*time.Second
	testingDirName         = "testing"
)

func setupTestLogger() {
	removeFiles()
	_ = os.Mkdir(testingDirName, 0700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      "testing.Log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)
}

func teardownTestLogger() {
	removeFiles()
}

func removeFiles() {
	_ = os.RemoveAll(testingDirName)
}

func setupTestDBConfig() configuration.InfluxDBConfig {
	return configuration.InfluxDBConfig{
		Database: "test",
		IP:       "http://localhost",
		Port:     "1234",
		User:     "user",
		Password: "password",
	}
}

func setupTestInflux(t *testing.T) (*db.Influx, *gomock.Controller, *mocks.MockClient) {
	setupTestLogger()

	ctl := gomock.NewController(t)
	mock := mocks.NewMockClient(ctl)
	return &db.Influx{
		Database: "test",
		Client:   mock,
		Log:      logger.New("test"),
		OK:       true,
	}, ctl, mock
}

func TestNewInfluxDBWriter(t *testing.T) {
	config := setupTestDBConfig()
	_, err := db.NewInfluxDBWriter(config, nil)

	assert.Equal(t, nil, err, "wrong error")
}

func TestAdd(t *testing.T) {
	i, ctl, _ := setupTestInflux(t)
	defer ctl.Finish()
	defer teardownTestLogger()

	data := db.InfluxData{
		Fields:      nil,
		Measurement: "measurement",
		Tags:        nil,
		Timing:      time.Time{},
	}

	assert.Equal(t, 0, len(i.Data), "wrong initial data size")

	i.Add(data)
	assert.Equal(t, 1, len(i.Data), "wrong data size")
	assert.Equal(t, data, i.Data[0], "wrong data")
}

func TestLoopWhenWriteNormal(t *testing.T) {
	i, ctl, mock := setupTestInflux(t)
	defer ctl.Finish()
	defer teardownTestLogger()

	mock.EXPECT().Write(gomock.Any()).Return(nil).Times(1)

	data := db.InfluxData{
		Fields:      map[string]interface{}{"name": "node"},
		Measurement: "measurement",
		Tags:        map[string]string{"value": "1.2"},
		Timing:      time.Time{},
	}

	i.Add(data)
	shutdownChan := make(chan struct{}, 1)
	go i.Loop(shutdownChan)

	timer := time.After(looperIntervalSecond)
	<-timer

	assert.Equal(t, 0, len(i.Data), "not cleanup after write to db")
}

func TestLoopWhenWriteError(t *testing.T) {
	i, ctl, mock := setupTestInflux(t)
	defer ctl.Finish()
	defer teardownTestLogger()

	mock.EXPECT().Write(gomock.Any()).Return(fmt.Errorf("error")).Times(1)

	data := db.InfluxData{
		Fields:      map[string]interface{}{"name": "node"},
		Measurement: "measurement",
		Tags:        map[string]string{"value": "1.2"},
		Timing:      time.Time{},
	}

	i.Add(data)
	shutdownChan := make(chan struct{}, 1)
	go i.Loop(shutdownChan)

	timer := time.After(looperIntervalSecond)
	<-timer

	assert.Equal(t, 1, len(i.Data), "not cleanup after write to db")
}
