package node

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
)

// Node - node interface
type Node interface {
	BroadcastReceiver() network.Client
	CommandSenderAndReceiver() network.Client
	CheckTimer() *time.Timer
	Client() Remote
	CloseConnection() error
	DropRate()
	Log() *logger.L
	Monitor(shutdown <-chan struct{})
	StopMonitor()
	Verify()
}

type node struct {
	config     configuration.NodeConfig
	client     Remote
	id         int
	log        *logger.L
	checkTimer *time.Timer
}

type nodeKeys struct {
	private      []byte
	public       []byte
	remotePublic []byte
}

const (
	receiveBroadcastIntervalInSecond = 120 * time.Second
	checkIntervalSecond              = 10 * time.Second
)

// NewNode - create new node
func NewNode(config configuration.NodeConfig, keys configuration.Keys, idx int) (intf Node, err error) {
	log := logger.New(fmt.Sprintf("node-%d", idx))

	n := &node{
		config:     config,
		id:         idx,
		log:        log,
		checkTimer: time.NewTimer(checkIntervalSecond),
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

func parseKeys(keys configuration.Keys, remotePublicKeyStr string) (*nodeKeys, error) {
	publicKey, err := network.ReadPublicKey(keys.Public)
	if nil != err {
		return nil, err
	}

	privateKey, err := network.ReadPrivateKey(keys.Private)
	if nil != err {
		return nil, err
	}

	remotePublicKey, err := hex.DecodeString(remotePublicKeyStr)
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

// BroadcastReceiverClient - get zmq broadcast receiver remote
func (n *node) BroadcastReceiver() network.Client {
	return n.client.BroadcastReceiver()
}

// CommandSenderAndReceiver - network remote of command sender and receiver
func (n *node) CommandSenderAndReceiver() network.Client {
	return n.client.CommandSenderAndReceiver()
}

// CheckTimer - get sender timer
func (n *node) CheckTimer() *time.Timer {
	return n.checkTimer
}

// Remote - return remote interface
func (n *node) Client() Remote {
	return n.client
}

// CloseConnection - close connection
func (n *node) CloseConnection() error {
	if err := n.client.Close(); nil != err {
		return err
	}
	return nil
}

// DropRate - drop rate
func (n *node) DropRate() {
	return
}

// Log - get logger
func (n *node) Log() *logger.L {
	return n.log
}

// Monitor - start to monitor
func (n *node) Monitor(shutdownCh <-chan struct{}) {
	go receiverLoop(n, shutdownCh, n.id)
	n.checkTimer.Reset(checkIntervalSecond)
	go checkerLoop(n, shutdownCh)

loop:
	select {
	case <-shutdownCh:
		break loop
	}

	n.Log().Info("stop")

	return
}

// StopMonitor - stop monitor
func (n *node) StopMonitor() {
	return
}

// Verify - verify record data
func (n *node) Verify() {
	return
}
