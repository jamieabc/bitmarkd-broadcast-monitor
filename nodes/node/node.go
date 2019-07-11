package node

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	zmq "github.com/pebbe/zmq4"
)

type Node interface {
	Client() *zmqutil.Client
	CloseConnection()
	DropRate()
	Log() *logger.L
	Monitor()
	setSenderAndReceiver(*zmq.Socket, *zmq.Socket)
	StopMonitor()
	StopReceiver()
	StopSender()
	Verify()
}

type NodeImpl struct {
	config   configuration.NodeConfig
	log      *logger.L
	client   *zmqutil.Client
	sender   *zmq.Socket
	receiver *zmq.Socket
}

const (
	receiveBroadcastInterval = 120 * time.Second
	signal                   = "inproc://bitmarkd-broadcast-monitor-signal"
)

// Initialise - initialise node
func Initialise(config configuration.NodeConfig, keys configuration.Keys, log *logger.L) (intf Node, err error) {
	n := &NodeImpl{
		config: config,
		log:    log,
	}

	publicKey, err := zmqutil.ReadPublicKey(keys.Public)
	if nil != err {
		log.Errorf("read public key: %q, error: %s", keys.Public, err)
		return nil, err
	}
	log.Debugf("read public key: %q", publicKey)

	privateKey, err := zmqutil.ReadPrivateKey(keys.Private)
	if nil != err {
		log.Errorf("read private key: %q, error: %s", keys.Private)
		return nil, err
	}
	log.Debugf("read private key: %q", privateKey)

	address, err := util.NewConnection(config.AddressIPv4)
	if nil != err {
		log.Errorf("node address: %q, error: %s", address, err)
		return nil, err
	}
	log.Debugf("new connection address: %s", address)

	serverPublicKey, err := hex.DecodeString(config.PublicKey)
	if nil != err {
		log.Errorf("node public key: %q, error: %s", config.PublicKey, err)
		return nil, err
	}
	log.Debugf("server public key: %q", serverPublicKey)

	if bytes.Equal(publicKey, serverPublicKey) {
		log.Errorf("node public key: %q, error: %s", publicKey, err)
		return nil, err
	}

	client, err := zmqutil.NewClient(zmq.SUB, privateKey, publicKey, 0)
	defer func() {
		if nil != err && nil != client {
			zmqutil.CloseClients([]*zmqutil.Client{client})
		}
	}()

	if nil != err {
		log.Errorf("node address: %q, error: %s", address, err)
		return nil, err
	}

	err = client.Connect(address, serverPublicKey, config.Chain)
	if nil != err {
		log.Errorf("node connect to %q, error: %s", address, err)
		return nil, err
	}
	log.Infof("connect remote %s", client.String())

	n.client = client

	return n, err
}

// Run - run go routines
func Run(n Node, shutdown <-chan struct{}) {
	go receiveLoop(n)

loop:
	select {
	case <-shutdown:
		break loop
	}

	log := n.Log()
	log.Info("stop")

	n.StopSender()
}

func receiveLoop(node Node) {
	log := node.Log()

	sender, receiver, err := zmqutil.NewSignalPair(signal)
	if nil != err {
		log.Errorf("create signal pair error: %s", err)
		return
	}
	node.setSenderAndReceiver(sender, receiver)

	poller := zmqutil.NewPoller()
	client := node.Client()
	_ = client.BeginPolling(poller, zmq.POLLIN)
	poller.Add(receiver, zmq.POLLIN)

loop:
	for {
		log.Info("waiting to receive broadcast...")
		polled, _ := poller.Poll(receiveBroadcastInterval)
		if 0 == len(polled) {
			log.Info("over heartbeat receive time")
			continue
		}
		for _, p := range polled {
			switch s := p.Socket; s {
			case receiver:
				_, err := s.RecvMessageBytes(0)
				if nil != err {
					log.Errorf("receive error: %s", err)
				}
				log.Debug("receive stop message")
				break loop
			default:
				data, err := s.RecvMessageBytes(0)
				log.Debugf("receive chain %s broadcast message", data[0])
				if nil != err {
					log.Errorf("receive error: %s", err)
					continue
				}
				process(node, data[1:], node.Client())
			}
		}
	}

	node.StopReceiver()
	node.CloseConnection()
	log.Flush()

	return
}

func process(node Node, data [][]byte, client *zmqutil.Client) {
	log := node.Log()
	log.Info("incomfing message")

	switch d := data[0]; string(d) {
	case "block":
		log.Debugf("receive block: %x", data[1])
		header, digest, _, err := blockrecord.ExtractHeader(data[1])
		if nil != err {
			log.Errorf("extract header with error: %s", err)
			return
		}
		log.Infof("receive block %d, previous block %s, digest: %s", header.Number, header.PreviousBlock.String(), digest.String())

	case "heart":
		log.Infof("receive heartbeat")
	default:
		log.Debugf("receive category: %s", d)
	}
}

func (n *NodeImpl) Client() *zmqutil.Client {
	return n.client
}

func (n *NodeImpl) CloseConnection() {
	n.client.Close()
}

func (n *NodeImpl) DropRate() {
	return
}

func (n *NodeImpl) Log() *logger.L {
	return n.log
}

func (n *NodeImpl) Monitor() {
	return
}

func (n *NodeImpl) setSenderAndReceiver(sender *zmq.Socket, receiver *zmq.Socket) {
	n.sender = sender
	n.receiver = receiver
}

func (n *NodeImpl) StopReceiver() {
	n.receiver.Close()
}

func (n *NodeImpl) StopSender() {
	n.log.Debug("stop sender")
	_, err := n.sender.SendMessage("stop")
	if nil != err {
		n.log.Errorf("send message with error: %s", err)
	}
	n.sender.Close()
}

func (n *NodeImpl) StopMonitor() {
	return
}

func (n *NodeImpl) Verify() {
	return
}
