package node

import (
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

func receiverLoop(n Node, shutdownCh <-chan struct{}, id int) {
	eventChannel := make(chan zmq.Polled, 10)
	log := n.Log()

	poller, err := network.NewPoller(eventChannel, shutdownCh, id)
	if nil != err {
		log.Errorf("create poller with error: %s", err)
		return
	}
	poller.Add(n.BroadcastReceiver(), zmq.POLLIN)

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
				process(n, data)
			}
		}
	}()

	<-shutdownCh

	if err := stopInternalSignalReceiver(); nil != err {
		log.Errorf("stop internal signal with error: %s", err)
	}
	if err := n.CloseConnection(); nil != err {
		log.Errorf("close connection with error: %s", err)
	}
	log.Flush()

	return
}

func process(n Node, data [][]byte) {
	chain := data[0]
	log := n.Log()

	switch d := data[1]; string(d) {
	case "block":
		log.Debugf("block: %x", data[2])
		header, digest, _, err := blockrecord.ExtractHeader(data[2])
		if nil != err {
			log.Errorf("extract header with error: %s", err)
			return
		}

		log.Infof("receive chain %s, block %d, previous block %s, digest: %s",
			chain,
			header.Number,
			header.PreviousBlock.String(),
			digest.String(),
		)

	case "heart":
		log.Infof("receive heartbeat")

	default:
		log.Infof("receive %s", d)
	}
}
