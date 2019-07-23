package node

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

type Node interface {
	BroadcastReceiver() *network.Client
	CloseConnection()
	CommandSenderAndReceiver() *network.Client
	DropRate()
	Log() *logger.L
	Monitor(<-chan struct{})
	StopMonitor()
	Verify()
}

type NodeImpl struct {
	config            configuration.NodeConfig
	client            NodeClient
	id                int
	log               *logger.L
	clientSenderTimer *time.Timer
}

type nodeKeys struct {
	private      []byte
	public       []byte
	remotePublic []byte
}

const (
	receiveBroadcastIntervalInSecond = 120 * time.Second
	senderCheckIntervalInSecond      = 10 * time.Second
	signal                           = "inproc://bitmarkd-broadcast-monitor-signal"
)

var (
	internalSignalSender   *zmq.Socket
	internalSignalReceiver *zmq.Socket
)

// Initialise - initialise node settings
func Initialise() error {
	var err error
	internalSignalSender, internalSignalReceiver, err = network.NewSignalPair(signal)
	if nil != err {
		logger.Panic("create signal pair error")
		return err
	}

	return err
}

// NewNode - create new node
func NewNode(config configuration.NodeConfig, keys configuration.Keys, idx int) (intf Node, err error) {
	log := logger.New(fmt.Sprintf("node-%d", idx))

	n := &NodeImpl{
		config:            config,
		id:                idx,
		log:               log,
		clientSenderTimer: time.NewTimer(time.Duration(senderCheckIntervalInSecond)),
	}

	nodeKey, err := parseKeys(keys, config.PublicKey)
	if nil != err {
		log.Errorf("parse keys: %v, remote public key: %q with error: %s", keys, config.PublicKey, err)
		return nil, err
	}

	log.Debugf("public key: %q, private key: %q, remote public key: %q", nodeKey.public, nodeKey.private, nodeKey.remotePublic)

	n.client, err = newClient(config, nodeKey)
	if nil != err {
		return nil, err
	}

	return n, nil
}

func parseKeys(keys configuration.Keys, remotePublickeyStr string) (*nodeKeys, error) {
	publicKey, err := network.ReadPublicKey(keys.Public)
	if nil != err {
		return nil, err
	}

	privateKey, err := network.ReadPrivateKey(keys.Private)
	if nil != err {
		return nil, err
	}

	remotePublicKey, err := hex.DecodeString(remotePublickeyStr)
	if nil != err {
		return nil, err
	}

	if bytes.Equal(publicKey, remotePublicKey) {
		return nil, fmt.Errorf("remote and local public key: %q same , error: %s", publicKey, err)
	}

	return &nodeKeys{
		private:      privateKey,
		public:       publicKey,
		remotePublic: remotePublicKey,
	}, nil
}

// BroadcastReceiverClient - get zmq broadcast receiver client
func (n *NodeImpl) BroadcastReceiver() *network.Client {
	return n.client.BroadcastReceiver()
}

func (n *NodeImpl) CommandSenderAndReceiver() *network.Client {
	return n.client.CommandSenderAndReceiver()
}

// CloseConnection - close connection
func (n *NodeImpl) CloseConnection() {
	n.client.Close()
}

// DropRate - drop rate
func (n *NodeImpl) DropRate() {
	return
}

// Log - get logger
func (n *NodeImpl) Log() *logger.L {
	return n.log
}

// Monitor - start to monitor
func (n *NodeImpl) Monitor(shutdown <-chan struct{}) {
	go n.receiverLoop()
	n.clientSenderTimer.Reset(senderCheckIntervalInSecond)
	go n.senderLoop(shutdown)

loop:
	select {
	case <-shutdown:
		break loop
	}

	n.Log().Info("stop")

	stopInternalGoRoutines()
	return
}

func (n *NodeImpl) receiverLoop() {
	poller := network.NewPoller()
	broadcastReceiver := n.BroadcastReceiver()
	_ = broadcastReceiver.BeginPolling(poller, zmq.POLLIN)
	poller.Add(internalSignalReceiver, zmq.POLLIN)

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
				n.process(data)
			}
		}
	}

	stopInternalSignalReceiver()
	n.CloseConnection()
	log.Flush()

	return
}

func (n *NodeImpl) process(data [][]byte) {
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

func stopInternalSignalReceiver() {
	internalSignalReceiver.Close()
}

func (n *NodeImpl) senderLoop(shutdown <-chan struct{}) {
	log := n.Log()
	for {
		select {
		case <-shutdown:
			log.Infof("receive shutdown signal")
			break
		case <-n.clientSenderTimer.C:
			log.Debug("get client info")
			info, err := n.client.Info()
			if nil != err {
				log.Errorf("get client info error: %s", err)
				log.Infof("client info: %v\n", info)
			}
			n.clientSenderTimer.Reset(senderCheckIntervalInSecond)
		}
	}
	log.Infof("finish")
}

func stopInternalGoRoutines() {
	_, err := internalSignalSender.SendMessage("stop")
	if nil != err {
		logger.Criticalf("send stop message with error: %s", err)
	}
	internalSignalSender.Close()
}

// StopMonitor - stop monitor
func (n *NodeImpl) StopMonitor() {
	return
}

// Verify - verify record data
func (n *NodeImpl) Verify() {
	return
}
