package node

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	zmq "github.com/pebbe/zmq4"
)

type Node interface {
	BroadcastReceiver() *zmqutil.Client
	CloseConnection()
	CommandSenderAndReceiver() *zmqutil.Client
	DropRate()
	Log() *logger.L
	Monitor()
	StopMonitor()
	Verify()
}

type NodeImpl struct {
	config configuration.NodeConfig
	client NodeClient
	id     int
	log    *logger.L
}

type nodeKeys struct {
	private      []byte
	public       []byte
	remotePublic []byte
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
	publicKey, err := zmqutil.ReadPublicKey(keys.Public)
	if nil != err {
		return nil, err
	}

	privateKey, err := zmqutil.ReadPrivateKey(keys.Private)
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

func broadcastAddressAndPort(config configuration.NodeConfig) string {
	return hostAndPort(config.AddressIPv4, config.BroadcastPort)
}

func commandAddressAndPort(config configuration.NodeConfig) string {
	return hostAndPort(config.AddressIPv4, config.CommandPort)
}

func hostAndPort(host string, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

// Run - run go routines
func Run(n Node, shutdown <-chan struct{}) {
	go receiverLoop(n)

loop:
	select {
	case <-shutdown:
		break loop
	}

	n.Log().Info("stop")

	stopSender()
}

func receiverLoop(node Node) {
	poller := zmqutil.NewPoller()
	broadcastReceiver := node.BroadcastReceiver()
	_ = broadcastReceiver.BeginPolling(poller, zmq.POLLIN)
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
				process(node, data, node.BroadcastReceiver())
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

func stopSender() {
	_, err := sender.SendMessage("stop")
	if nil != err {
		logger.Criticalf("send stop message with error: %s", err)
	}
	sender.Close()
}

func process(node Node, data [][]byte, broadcastReceiver *zmqutil.Client) {
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

		log.Infof("receive chain %s, block %d, previous block %s, digest: %s",
			chain,
			header.Number,
			header.PreviousBlock.String(),
			digest.String(),
		)

	case "heart":
		log.Infof("receive heartbeat")

	default:
		log.Debugf("receive %s", d)
	}
}

// BroadcastReceiverClient - get zmq broadcast receiver client
func (n *NodeImpl) BroadcastReceiver() *zmqutil.Client {
	return n.client.BroadcastReceiver()
}

func (n *NodeImpl) CommandSenderAndReceiver() *zmqutil.Client {
	return n.client.CommandSenderAndReceiver()
}

// CloseConnection - close connection
func (n *NodeImpl) CloseConnection() {
	n.client.CloseBroadcastReceiver()
	n.client.CloseCommandSenderAndReceiver()
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
func (n *NodeImpl) Monitor() {
	return
}

// StopMonitor - stop monitor
func (n *NodeImpl) StopMonitor() {
	return
}

// Verify - verify record data
func (n *NodeImpl) Verify() {
	return
}
