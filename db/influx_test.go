package db_test

import (
	"testing"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db/mocks"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
)

const (
	originalIntervalSecond = 10 * time.Second
	looperIntervalSecond   = originalIntervalSecond + 1*time.Second
)

func setupTestDBConfig() configuration.InfluxDBConfig {
	return configuration.InfluxDBConfig{
		Database: "test",
		IPv4:     "http://localhost",
		Port:     "1234",
		User:     "user",
		Password: "password",
	}
}

func setupTestInflux(t *testing.T) (*db.Influx, *gomock.Controller, *mocks.MockClient) {
	ctl := gomock.NewController(t)
	mock := mocks.NewMockClient(ctl)
	return &db.Influx{
		Database: "test",
		Client:   mock,
	}, ctl, mock
}

func TestNewInfluxDBWriter(t *testing.T) {
	config := setupTestDBConfig()
	_, err := db.NewInfluxDBWriter(config)

	assert.Equal(t, nil, err, "wrong error")
}

func TestAdd(t *testing.T) {
	i, ctl, _ := setupTestInflux(t)
	defer ctl.Finish()

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

func TestLoop(t *testing.T) {
	i, ctl, mock := setupTestInflux(t)
	defer ctl.Finish()

	data := db.InfluxData{
		Fields:      map[string]interface{}{"name": "node"},
		Measurement: "measurement",
		Tags:        map[string]string{"value": "1.2"},
		Timing:      time.Time{},
	}

	mock.EXPECT().Write(gomock.Any()).Return(nil).Times(1)
	mock.EXPECT().Close().Return(nil).Times(1)

	i.Add(data)
	shutdownChan := make(chan struct{}, 1)
	go i.Loop(shutdownChan)

	timer := time.After(looperIntervalSecond)
	<-timer

	shutdownChan <- struct{}{}
}
