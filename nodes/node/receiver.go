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
	assetCmdStr                      = "assets"
	issueCmdStr                      = "issues"
	transferCmdStr                   = "transfer"
	blockCmdStr                      = "block"
	heartbeatCmdStr                  = "heart"
	checkTimeSecond                  = 60 * time.Second
	receiveBroadcastIntervalInSecond = 120 * time.Second
	eventChannelSize                 = 100
)

func receiverLoop(n Node, rs recorders, id int) {
	log := n.Log()
	timer := clock.NewClock()

	go rs.heartbeat.RemoveOutdatedPeriodically(timer)
	go rs.transaction.RemoveOutdatedPeriodically(timer)
	go receiverRoutine(n, rs, id)

	<-shutdownChan

	if err := n.CloseConnection(); nil != err {
		log.Errorf("close connection with error: %s", err)
	}
	log.Flush()

	return
}

func receiverRoutine(n Node, rs recorders, id int) {
	eventChan := make(chan zmq.Polled, eventChannelSize)
	log := n.Log()
	checkTimer := time.After(checkTimeSecond)
	checked := false

	poller, err := initializePoller(n, id, eventChan)
	if nil != err {
		log.Errorf("initialize poller with error: %s", err)
		return
	}

	for {
		_ = poller.Start(receiveBroadcastIntervalInSecond)
		select {
		case polled := <-eventChan:
			data, err := polled.Socket.RecvMessageBytes(0)
			if nil != err {
				log.Errorf("receive message with error: %s", err)
				continue
			}
			process(n, rs, data, &checked)
		case <-shutdownChan:
			log.Infof("terminate receiver loop")
			return
		case <-checkTimer:
			checked = false
			checkTimer = time.After(checkTimeSecond)
		}
	}
}

func initializePoller(n Node, id int, eventChan chan zmq.Polled) (network.Poller, error) {
	poller, err := network.NewPoller(eventChan, shutdownChan, id)
	if nil != err {
		return nil, err
	}
	poller.Add(n.BroadcastReceiver(), zmq.POLLIN)
	return poller, nil
}

func process(n Node, rs recorders, data [][]byte, checked *bool) {
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
		rs.transaction.Add(now, id)
		if !*checked {
			*checked = true
			notifyChan <- struct{}{}
		}

	case heartbeatCmdStr:
		log.Infof("receive heartbeat")
		rs.heartbeat.Add(now)

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
