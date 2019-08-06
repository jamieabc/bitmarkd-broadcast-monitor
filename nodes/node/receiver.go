package node

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/clock"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

const (
	assetCategoryStr    = "assetCategoryStr"
	issueCategoryStr    = "issueCategoryStr"
	transferCategoryStr = "transferCategoryStr"
)

func receiverLoop(n Node, rs recorders, shutdownCh <-chan struct{}, id int) {
	eventChannel := make(chan zmq.Polled, 10)
	log := n.Log()

	poller, err := network.NewPoller(eventChannel, shutdownCh, id)
	if nil != err {
		log.Errorf("create poller with error: %s", err)
		return
	}
	poller.Add(n.BroadcastReceiver(), zmq.POLLIN)
	timer := clock.NewClock()

	go rs.heartbeat.CleanupPeriodically(timer)
	go rs.transaction.CleanupPeriodically(timer)

	go func() {
		for {
			_ = poller.Start(receiveBroadcastIntervalInSecond)
			log.Debug("waiting broadcast...")

			select {
			case polled := <-eventChannel:
				data, err := polled.Socket.RecvMessageBytes(0)
				if nil != err {
					log.Errorf("receive error: %s", err)
					continue
				}
				process(n, rs, data)
			case <-shutdownCh:
				return
			}
		}
	}()

	<-shutdownCh

	if err := n.CloseConnection(); nil != err {
		log.Errorf("close connection with error: %s", err)
	}
	log.Flush()

	return
}

func process(n Node, rs recorders, data [][]byte) {
	log := n.Log()
	blockchain := string(data[0])
	if !chain.Valid(blockchain) {
		log.Errorf("invalid chain: %s", blockchain)
		return
	}
	now := time.Now()

	switch category := string(data[1]); category {
	case assetCategoryStr, issueCategoryStr, transferCategoryStr:
		trx := data[2]

		log.Infof("receive %s record", category)
		log.Debugf("transactions: %x", trx)

		txID, err := transactionID(trx, blockchain, log)
		if nil != err {
			return
		}
		log.Infof("transaction ID: %s", txID)
		rs.transaction.Add(now, txID)

	case "heart":
		log.Infof("receive heartbeat")
		rs.heartbeat.Add(now)

	default:
		log.Infof("receive %s", category)
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
