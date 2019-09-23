package node

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"

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
	pollerTimeoutSecond       = 30 * time.Second
	transactionTimeoutSecond  = 120 * time.Second
	eventChannelSize          = 100
	reconnectDelayMillisecond = 20 * time.Millisecond
	keyLength                 = 10
)

func receiverLoop(n Node, rs recorders, id int) {
	log := n.Log()
	timer := clock.NewClock()

	go rs.transaction.PeriodicRemove(timer)
	go rs.block.PeriodicRemove(timer)
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
	transactionTimer := time.NewTimer(transactionTimeoutSecond)

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
			process(n, rs, data)
			resetTimer(transactionTimer, log)

		case <-shutdownChan:
			log.Infof("terminate receiver loop")
			return

		case <-transactionTimer.C:
			log.Warn("transaction timeout exceed, reset timer")
			//reconnect(poller, n, transactionTimer)
			resetTimer(transactionTimer, log)
		}
	}
}

func resetTimer(t *time.Timer, log *logger.L) {
	log.Debug("reset transaction timer")
	if !t.Stop() {
		<-t.C
	}
	ok := t.Reset(transactionTimeoutSecond)
	if ok {
		log.Warn("transaction timer still active")
	}
}

//sometimes not receiving transaction for some time, then need to close the socket and open a new one
func reconnect(poller network.Poller, n Node, transactionTimer *time.Timer) {
	log := n.Log()

	log.Info("closing broadcast receiver connection")
	poller.Remove(n.BroadcastReceiver())
	err := n.BroadcastReceiver().Reconnect()
	if nil != err {
		n.Log().Errorf("reconnect with error: %s, abort", err)
		return
	}
	time.Sleep(reconnectDelayMillisecond)
	log.Infof("adding socket %s to poller", n.BroadcastReceiver().String())
	poller.Add(n.BroadcastReceiver(), zmq.POLLIN)
	log.Debug("reset transaction timer")
	transactionTimer.Reset(transactionTimeoutSecond)
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

func process(n Node, rs recorders, data [][]byte) {
	log := n.Log()
	blockchain := string(data[0])
	if !chain.Valid(blockchain) {
		log.Errorf("invalid chain: %s", blockchain)
		return
	}
	now := time.Now()

	switch category := string(data[1]); category {
	case blockCmdStr:
		key := string(data[2][0:keyLength])

		value, found := caches.Get(key)
		var header blockrecord.Header
		if !found {
			ptr, _, _, err := blockrecord.ExtractHeader(data[2], uint64(0))
			if nil != err {
				log.Errorf("extract block header with error: %s", err)
				return
			}
			header = *ptr
			caches.Set(key, header)
		} else {
			header = value.(blockrecord.Header)
		}

		log.Infof("receive block %d", header.Number)
		rs.block.Add(now, recorder.BlockData{
			Number:       header.Number,
			GenerateTime: time.Unix(int64(header.Timestamp), 0),
		})

	case assetCmdStr, issueCmdStr, transferCmdStr:
		log.Debugf("raw %s data: %s", category, string(data[2]))
		key := hex.EncodeToString(data[2])[0:keyLength]

		var err error
		var id merkle.Digest
		value, found := caches.Get(key)
		if !found {
			if id, err = extractID(data[2], blockchain, log); nil != err {
				return
			}
			caches.Set(key, id)
		} else {
			id = value.(merkle.Digest)
		}

		log.Infof("receive %s broadcast, ID %s", category, []byte(fmt.Sprintf("%v", id)))

	case heartbeatCmdStr:
		log.Infof("receive heartbeat")

	default:
		log.Debugf("receive %s", category)
	}
	rs.transaction.Add(now)
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
