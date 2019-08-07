package node

import (
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
	assetCmdStr     = "assets"
	issueCmdStr     = "issues"
	transferCmdStr  = "transfer"
	blockCmdStr     = "block"
	heartbeatCmdStr = "heart"
)

func receiverLoop(n Node, rs recorders, id int) {
	eventChannel := make(chan zmq.Polled, 10)
	log := n.Log()

	poller, err := network.NewPoller(eventChannel, shutdownChan, id)
	if nil != err {
		log.Errorf("create poller with error: %s", err)
		return
	}
	poller.Add(n.BroadcastReceiver(), zmq.POLLIN)
	timer := clock.NewClock()
	checkTimer := time.After(30 * time.Second)
	checked := false

	go rs.heartbeat.CleanupPeriodically(timer)
	go rs.transaction.CleanupPeriodically(timer)

	go func() {
		for {
			_ = poller.Start(receiveBroadcastIntervalInSecond)
			select {
			case polled := <-eventChannel:
				data, err := polled.Socket.RecvMessageBytes(0)
				if nil != err {
					log.Errorf("receive error: %s", err)
					continue
				}
				process(n, rs, data, &checked)
			case <-shutdownChan:
				return
			case <-checkTimer:
				checked = false
				checkTimer = time.After(30 * time.Second)
			}
		}
	}()

	<-shutdownChan

	if err := n.CloseConnection(); nil != err {
		log.Errorf("close connection with error: %s", err)
	}
	log.Flush()

	return
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
		trx := data[2]

		txID, err := transactionID(trx, blockchain, log)
		if nil != err {
			return
		}
		log.Infof("receive %s: transaction ID %s", category, txID)
		rs.transaction.Add(now, txID)
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

func transactionID(trx []byte, chain string, log *logger.L) (merkle.Digest, error) {
	_, n, err := transactionrecord.Packed(trx).Unpack(isTestnet(chain))
	if nil != err {
		log.Errorf("unpack transaction with error: %s", err)
		return merkle.Digest{}, err
	}
	txID := transactionrecord.Packed(trx[:n]).MakeLink()
	return txID, nil
}

func isTestnet(category string) bool {
	return chain.Bitmark != category
}
