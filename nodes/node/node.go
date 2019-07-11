package node

import (
	"bytes"
	"encoding/hex"
	"fmt"
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
	StopMonitor()
	Verify()
}

type NodeImpl struct {
	config configuration.NodeConfig
	client *zmqutil.Client
	id     int
	log    *logger.L
}

const (
	receiveBroadcastInterval = 120 * time.Second
	signal                   = "inproc://bitmarkd-broadcast-monitor-signal"
)

var (
	sender   *zmq.Socket
	receiver *zmq.Socket
)

// Initialise - initialise node settings
func Initialise() error {
	var err error
	sender, receiver, err = zmqutil.NewSignalPair(signal)
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
		config: config,
		id:     idx,
		log:    log,
	}

	publicKey, privateKey, err := parseKeys(keys)
	if nil != err {
		log.Errorf("parse keys: %v with error: %s", keys, err)
		return nil, err
	}

	log.Debugf("public key: %q, private key: %q", publicKey, privateKey)

	address, err := util.NewConnection(config.AddressIPv4)
	if nil != err {
		log.Errorf("node address: %q, error: %s", address, err)
		return nil, err
	}
	log.Debugf("new connection address: %s", address)

	remotePublicKey, err := hex.DecodeString(config.PublicKey)
	if nil != err {
		log.Errorf("remote public key: %q, error: %s", config.PublicKey, err)
		return nil, err
	}
	log.Debugf("server public key: %q", remotePublicKey)

	if bytes.Equal(publicKey, remotePublicKey) {
		log.Errorf("remote and local public key: %q same , error: %s", publicKey, err)
		return nil, err
	}

	client, err := zmqutil.NewClient(zmq.SUB, privateKey, publicKey, 0)
	if nil != err {
		log.Errorf("node address: %q, error: %s", address, err)
		return nil, err
	}
	defer func() {
		if nil != err && nil != client {
			zmqutil.CloseClients([]*zmqutil.Client{client})
		}
	}()

	err = client.Connect(address, remotePublicKey, config.Chain)
	if nil != err {
		log.Errorf("node connect to %q, error: %s", address, err)
		return nil, err
	}
	log.Infof("connect remote %s", client.String())

	n.client = client

	return n, nil
}

func parseKeys(keys configuration.Keys) ([]byte, []byte, error) {
	publicKey, err := zmqutil.ReadPublicKey(keys.Public)
	if nil != err {
		return nil, nil, err
	}

	privateKey, err := zmqutil.ReadPrivateKey(keys.Private)
	if nil != err {
		return nil, nil, err
	}

	return publicKey, privateKey, nil
}

// Run - run go routines
func Run(n Node, shutdown <-chan struct{}) {
	go receiveLoop(n)

loop:
	select {
	case <-shutdown:
		break loop
	}

	n.Log().Info("stop")

	stopSender()
}

func stopSender() {
	_, err := sender.SendMessage("stop")
	if nil != err {
		logger.Criticalf("send stop message with error: %s", err)
	}
	sender.Close()
}

func receiveLoop(node Node) {
	poller := zmqutil.NewPoller()
	client := node.Client()
	_ = client.BeginPolling(poller, zmq.POLLIN)
	poller.Add(receiver, zmq.POLLIN)

	log := node.Log()

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
				if nil != err {
					log.Errorf("receive error: %s", err)
					continue
				}
				process(node, data, node.Client())
			}
		}
	}

	stopReceiver()
	node.CloseConnection()
	log.Flush()

	return
}

func stopReceiver() {
	receiver.Close()
}

func process(node Node, data [][]byte, client *zmqutil.Client) {
	chain := data[0]
	log := node.Log()

	switch d := data[1]; string(d) {
	case "block":
		log.Debugf("block: %x", data[2])
		header, digest, _, err := blockrecord.ExtractHeader(data[2])
		if nil != err {
			log.Errorf("extract header with error: %s", err)
			return
		}
		log.Infof("receive chain %s, block %d, previous block %s, digest: %s", chain, header.Number, header.PreviousBlock.String(), digest.String())

	case "heart":
		log.Infof("receive heartbeat")

	default:
		log.Debugf("receive %s", d)
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

func (n *NodeImpl) StopMonitor() {
	return
}

func (n *NodeImpl) Verify() {
	return
}
