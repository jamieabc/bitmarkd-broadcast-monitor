package node

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockrecord"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

const (
	assetCmdStr               = "assets"
	issueCmdStr               = "issues"
	transferCmdStr            = "transfer"
	blockCmdStr               = "block"
	heartbeatCmdStr           = "heart"
	checkTimeSecond           = 60 * time.Second
	pollerTimeoutSecond       = 30 * time.Second
	heartbeatTimeoutSecond    = 90 * time.Second
	eventChannelSize          = 100
	reconnectDelayMillisecond = 20 * time.Millisecond
)

func receiverLoop(n Node, rs recorders, id int) {
	log := n.Log()
	timer := clock.NewClock()

	go rs.heartbeat.RemoveOutdatedPeriodically(timer)
	//go rs.transaction.RemoveOutdatedPeriodically(timer)
	go receiverRoutine(n, rs, id)

	<-shutdownChan

	if err := n.Close(); nil != err {
		log.Errorf("close connection with error: %s", err)
	}
	log.Flush()

	return
}

func receiverRoutine(n Node, rs recorders, id int) {
	eventChan := make(chan zmq.Polled, eventChannelSize)
	log := n.Log()
	checkTimer := time.NewTimer(checkTimeSecond)
	heartbeatTimer := time.NewTimer(heartbeatTimeoutSecond)
	resetTimer := false
	checked := false

	poller, err := initialisePoller(n, id, eventChan)
	if nil != err {
		log.Errorf("initialise poller with error: %s", err)
		return
	}

	go poller.Start(pollerTimeoutSecond)

	for {
		log.Debug("waiting events...")
		select {
		case polled := <-eventChan:
			data, err := polled.Socket.RecvMessageBytes(-1)
			if nil != err {
				log.Errorf("receive message with error: %s", err)
				//error might comes from reopen socket, will behave normal after some retries
				continue
			}
			process(n, rs, data, &resetTimer, &checked)

		case <-shutdownChan:
			log.Infof("terminate receiver loop")
			return

		case <-checkTimer.C:
			checked = false
			checkTimer.Reset(checkTimeSecond)

		case <-heartbeatTimer.C:
			log.Warn("heartbeat timeout exceed, reopen heartbeat socket")
			go reconnect(poller, n, heartbeatTimer)
		}

		if resetTimer {
			log.Debug("reset heartbeat timeout timer")
			if !heartbeatTimer.Stop() {
				log.Debug("clear heartbeat timer channel")
				<-heartbeatTimer.C
				log.Debug("heartbeat timer channel cleared")
			}
			ok := heartbeatTimer.Reset(heartbeatTimeoutSecond)
			if ok {
				resetTimer = false
				log.Warn("heartbeat timer still active")
			}
			resetTimer = false
		}
	}
}

//sometimes not receives heartbeat for some time, then need to close the socket and open a new one
func reconnect(poller network.Poller, n Node, heartbeatTimer *time.Timer) {
	log := n.Log()

	log.Info("closing heartbeat socket")
	poller.Remove(n.BroadcastReceiver())
	err := n.BroadcastReceiver().Reconnect()
	if nil != err {
		n.Log().Errorf("reconnect with error: %s, abort", err)
		return
	}
	log.Infof("adding heartbeat socket %s to poller", n.BroadcastReceiver().String)
	poller.Add(n.BroadcastReceiver(), zmq.POLLIN)
	time.Sleep(reconnectDelayMillisecond)
	log.Debug("reset heartbeat timer")
	heartbeatTimer.Reset(heartbeatTimeoutSecond)
}

func initialisePoller(n Node, id int, eventChan chan zmq.Polled) (network.Poller, error) {
	pollerShutdownChan := make(chan struct{}, 1)
	poller, err := network.NewPoller(eventChan, pollerShutdownChan, id)
	if nil != err {
		return nil, err
	}
	poller.Add(n.BroadcastReceiver(), zmq.POLLIN)
	return poller, nil
}

func process(n Node, rs recorders, data [][]byte, resetTimer *bool, checked *bool) {
	log := n.Log()
	blockchain := string(data[0])
	if !chain.Valid(blockchain) {
		log.Errorf("invalid chain: %s", blockchain)
		return
	}
	now := time.Now()

	switch category := string(data[1]); category {
	case blockCmdStr:
		header, digest, _, err := blockrecord.ExtractHeader(data[2])
		if nil != err {
			log.Errorf("extract block header with error: %s", err)
			return
		}
		log.Infof("receive block, number: %d, digest: %s", header.Number, digest)

	case assetCmdStr, issueCmdStr, transferCmdStr:
		bytes := data[2]

		log.Debugf("raw %s data: %s", category, hex.EncodeToString(bytes))
		id, err := extractID(bytes, blockchain, log)
		if nil != err {
			return
		}
		log.Infof("receive %s ID %s", category, []byte(fmt.Sprintf("%v", id)))
		//rs.transaction.Add(now, id)
		//if !*checked {
		//	*checked = true
		//	notifyChan <- struct{}{}
		//}

	case heartbeatCmdStr:
		log.Infof("receive heartbeat")
		rs.heartbeat.Add(now)
		*resetTimer = true

	default:
		log.Debugf("receive %s", category)
	}
}

func extractID(bytes []byte, chain string, log *logger.L) (merkle.Digest, error) {
	_, n, err := transactionrecord.Packed(bytes).Unpack(isTestnet(chain))
	if nil != err {
		log.Errorf("unpack transaction with error: %s", err)
		return merkle.Digest{}, err
	}
	return transactionrecord.Packed(bytes[:n]).MakeLink(), nil
}

func isTestnet(category string) bool {
	return chain.Bitmark != category
}
