package node

import (
	"fmt"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

const (
	transactionCheckMinute = 2 * time.Minute
	blockCheckMinute       = 2 * time.Minute
	measurement            = "transaction-droprate"
)

// checkerLoop - loop to check all summaries
func checkerLoop(args []interface{}) {
	if 2 != len(args) {
		fmt.Println("checkerLoop wrong argument length")
		return
	}
	n := args[0].(Node)
	rs := args[1].(recorders)
	log := n.Log()
	transactionTimer := time.NewTimer(transactionCheckMinute)
	blockTimer := time.NewTimer(blockCheckMinute)

	for {
		select {
		case <-ctx.Done():
			log.Info("terminate checker loop")
			return

		case <-transactionTimer.C:
			ts := rs.transaction.Summary().(*recorder.TransactionSummary)

			writeToInfluxDB(ts, n.Name())

			log.Infof("transaction summary: %s", ts)
			transactionTimer.Reset(transactionCheckMinute)

		case <-blockTimer.C:
			bs := rs.block.Summary().(*recorder.BlocksSummary)
			if !bs.Valid() {
				sendToSlack(n.Name(), bs.String())
			}
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
