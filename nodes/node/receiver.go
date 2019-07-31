package node

import (
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

func receiverLoop(n Node) {
	poller := network.NewPoller()
	broadcastReceiver := n.BroadcastReceiver()
	poller.Add(broadcastReceiver, zmq.POLLIN)

	log := n.Log()

loop:
	for {
		log.Debug("waiting to receive broadcast...")
		polled, _ := poller.Poll(receiveBroadcastIntervalInSecond)
		if 0 == len(polled) {
			log.Info("over heartbeat receive time")
			continue
		}
		for _, p := range polled {
			switch s := p.Socket; s {
			case internalSignalReceiver:
				_, err := s.RecvMessageBytes(0)
				if nil != err {
					log.Errorf("receive error: %s", err)
				}
				log.Debug("receive stop message")
				break loop
			default:
				data, err := s.RecvMessageBytes(0)
				if nil != err {
					log.Errorf("receive error: %s", err)
					continue
				}
				process(n, data)
			}
		}
	}

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
