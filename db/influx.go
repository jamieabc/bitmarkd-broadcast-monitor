package db

import (
	"sync"
	"time"

	"github.com/bitmark-inc/logger"

	_ "github.com/influxdata/influxdb1-client"
	dbClient "github.com/influxdata/influxdb1-client/v2"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
)

const (
	dataSize             = 100
	looperIntervalSecond = 5 * time.Second
)

//InfluxData - data write to influx db
type InfluxData struct {
	Fields      map[string]interface{}
	Measurement string
	Tags        map[string]string
	Timing      time.Time
}

//Influx - influx db Data structure
type Influx struct {
	sync.Mutex
	Client   dbClient.Client
	Database string
	Data     []InfluxData
	Log      *logger.L
}

//Close - close Influx db connection
func (i *Influx) Close() error {
	if err := i.Client.Close(); nil != err {
		return err
	}
	return nil
}

//Add - set Fields and Tags
func (i *Influx) Add(data InfluxData) {
	i.Lock()
	i.Data = append(i.Data, data)
	i.Unlock()
}

//Loop - background loop
func (i *Influx) Loop(shutdownChan chan struct{}) {
	timer := time.After(looperIntervalSecond)
	for {
		select {
		case <-shutdownChan:
			_ = i.Close()
			return
		case <-timer:
			err := i.write()
			if nil != err {
				i.Log.Errorf("write to influx db with error: %s", err)
			}
			timer = time.After(looperIntervalSecond)
		}
	}
}

func (i *Influx) write() (err error) {
	if 0 == len(i.Data) {
		return
	}

	bp, err := dbClient.NewBatchPoints(dbClient.BatchPointsConfig{
		Database: i.Database,
	})

	if nil != err {
		return
	}

	points := make([]InfluxData, len(i.Data))

	i.Lock()
	copy(points, i.Data[:])
	i.Data = i.Data[:0]
	i.Unlock()

	var pt *dbClient.Point
	for _, d := range points {
		pt, err = dbClient.NewPoint(d.Measurement, d.Tags, d.Fields, d.Timing)
		if nil != err {
			i.Lock()
			i.Data = append(i.Data, points...)
			i.Unlock()
			return
		}

		bp.AddPoint(pt)
	}

	if err = i.Client.Write(bp); nil != err {
		i.Lock()
		i.Data = append(i.Data, points...)
		i.Unlock()
		return
	}

	return
}

//NewInfluxDBWriter - create Influx dbClient writer
func NewInfluxDBWriter(config configuration.InfluxDBConfig, log *logger.L) (DBWriter, error) {
	c, err := dbClient.NewHTTPClient(dbClient.HTTPConfig{
		Addr:               config.IPv4,
		Username:           config.User,
		Password:           config.Password,
		UserAgent:          "",
		Timeout:            0,
		InsecureSkipVerify: false,
		TLSConfig:          nil,
		Proxy:              nil,
	})

	if nil != err {
		return nil, err
	}

	return &Influx{
		Client:   c,
		Database: config.Database,
		Data:     make([]InfluxData, dataSize),
		Log:      log,
	}, nil
}
