package node

import (
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"
)

const (
	checkInterval = 1 * time.Minute
)

func checkerLoop(n Node, rs recorders, shutdownCh <-chan struct{}) {
	log := n.Log()
	timer := time.After(checkInterval)

loop:
	for {
		select {
		case <-shutdownCh:
			break loop
		case <-timer:
			log.Infof("heartbeat summary: %s", rs.heartbeat.Summary().(*recorder.HeartbeatSummary))
			log.Infof("transaction summary: %s", rs.transaction.Summary().(*recorder.TransactionSummary))
			timer = time.After(checkInterval)
		}
	}
}
