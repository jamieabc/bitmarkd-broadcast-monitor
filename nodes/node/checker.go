package node

import (
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

const (
	checkInterval = 1 * time.Minute
)

func checkerLoop(n Node, rs recorders) {
	log := n.Log()
	timer := time.After(checkInterval)

loop:
	for {
		select {
		case <-shutdownChan:
			log.Info("terminate checker loop")
			break loop
		case <-timer:
			log.Infof("heartbeat summary: %s", rs.heartbeat.Summary().(*recorder.HeartbeatSummary))
			log.Infof("transaction summary: %s", rs.transaction.Summary().(*recorder.TransactionSummary))
			timer = time.After(checkInterval)
		}
	}
}
