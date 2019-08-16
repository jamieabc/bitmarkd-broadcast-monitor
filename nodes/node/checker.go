package node

import (
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

const (
	checkInterval = 1 * time.Minute
	measurement   = "heartbeat-droprate"
)

func checkerLoop(n Node, rs recorders) {
	log := n.Log()
	timer := time.After(checkInterval)

	for {
		select {
		case <-shutdownChan:
			log.Info("terminate checker loop")
			return

		case <-timer:
			hs := rs.heartbeat.Summary().(*recorder.HeartbeatSummary)
			ts := rs.transaction.Summary().(*recorder.TransactionSummary)

			writeToInfluxDB(hs, n.Name())

			log.Infof("heartbeat summary: %s", hs)
			log.Infof("transaction summary: %s", ts)
			timer = time.After(checkInterval)
		}
	}
}

func writeToInfluxDB(sum *recorder.HeartbeatSummary, name string) {
	db.Add(db.InfluxData{
		Fields:      map[string]interface{}{"value": sum.Droprate},
		Measurement: measurement,
		Tags:        map[string]string{"name": name},
		Timing:      time.Now(),
	})
}
