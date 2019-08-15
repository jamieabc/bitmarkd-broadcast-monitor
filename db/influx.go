package db

import (
	"time"

	_ "github.com/influxdata/influxdb1-client"
	dbClient "github.com/influxdata/influxdb1-client/v2"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
)

type influx struct {
	client   dbClient.Client
	database string
	tags     map[string]string
	fields   map[string]interface{}
}

//Close - close influx db connection
func (i *influx) Close() error {
	if err := i.client.Close(); nil != err {
		return err
	}
	return nil
}

//Fields - set fields, not thread safe
func (i *influx) Fields(fields map[string]interface{}) {
	i.fields = fields
}

//Tags - set tags, not thread safe
func (i *influx) Tags(tags map[string]string) {
	i.tags = tags
}

//Write - write to influx db, not thread safe
func (i *influx) Write(measurement []byte) (n int, err error) {
	n = 0
	defer func() {
		err = i.Close()
		return
	}()

	bp, err := dbClient.NewBatchPoints(dbClient.BatchPointsConfig{
		Precision:        "",
		Database:         i.database,
		RetentionPolicy:  "",
		WriteConsistency: "",
	})

	if nil != err {
		return
	}

	pt, err := dbClient.NewPoint(string(measurement), i.tags, i.fields, time.Now())
	if nil != err {
		return
	}

	bp.AddPoint(pt)

	if err := i.client.Write(bp); nil != err {
		return
	}

	n = 1
	return
}

//NewInfluxDBWriter - create influx dbClient writer
func NewInfluxDBWriter(config configuration.InfluxDBConfig) (DBWriter, error) {
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

	return &influx{
		client:   c,
		database: config.Database,
	}, nil
}
