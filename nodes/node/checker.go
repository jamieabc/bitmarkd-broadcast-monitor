package node

import (
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

const (
	transactionCheckMinute = 2 * time.Minute
	blockCheckMinute       = 2 * time.Minute
	measurement            = "transaction-droprate"
)

func checkerLoop(n Node, rs recorders) {
	log := n.Log()
	transactionTimer := time.NewTimer(transactionCheckMinute)
	blockTimer := time.NewTimer(blockCheckMinute)

	for {
		select {
		case <-shutdownChan:
			log.Info("terminate checker loop")
			return

		case <-transactionTimer.C:
			//hs := rs.heartbeat.Summary().(*recorder.HeartbeatSummary)
			ts := rs.transaction.Summary().(*recorder.TransactionSummary)

			writeToInfluxDB(ts, n.Name())

			//log.Infof("heartbeat summary: %s", hs)
			log.Infof("transaction summary: %s", ts)
			transactionTimer.Reset(transactionCheckMinute)

		case <-blockTimer.C:
			bs := rs.block.Summary().(*recorder.BlocksSummary)
			log.Infof("block summary: %s", bs)
			blockTimer.Reset(blockCheckMinute)
		}
	}
}

func writeToInfluxDB(sum *recorder.TransactionSummary, name string) {
	db.Add(db.InfluxData{
		Fields:      map[string]interface{}{"value": sum.Droprate},
		Measurement: measurement,
		Tags:        map[string]string{"name": name},
		Timing:      time.Now(),
	})
}
