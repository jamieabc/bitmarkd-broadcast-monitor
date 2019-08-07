package node

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
)

//Node - node interface
type Node interface {
	BroadcastReceiver() network.Client
	CommandSenderAndReceiver() network.Client
	CheckTimer() *time.Timer
	Client() Remote
	CloseConnection() error
	DropRate()
	Log() *logger.L
	Monitor()
	StopMonitor()
	Verify()
}

type recorders struct {
	heartbeat   recorder.Recorder
	transaction recorder.Recorder
}

type node struct {
	checkTimer          *time.Timer
	client              Remote
	config              configuration.NodeConfig
	heartbeatRecorder   recorder.Recorder
	id                  int
	log                 *logger.L
	transactionRecorder recorder.Recorder
}

type nodeKeys struct {
	private      []byte
	public       []byte
	remotePublic []byte
}

const (
	checkIntervalSecond = 2 * time.Minute
)

var (
	shutdownChan <-chan struct{}
	notifyChan   chan struct{}
)

//Initialise
func Initialise(shutdown <-chan struct{}) {
	shutdownChan = shutdown
	notifyChan = make(chan struct{}, 1)
	recorder.Initialise(shutdown)
}

//NewNode - create new node
func NewNode(config configuration.NodeConfig, keys configuration.Keys, idx int, heartbeatIntervalSecond int) (intf Node, err error) {
	log := logger.New(fmt.Sprintf("node-%d", idx))

	n := &node{
		checkTimer:          time.NewTimer(checkIntervalSecond),
		config:              config,
		heartbeatRecorder:   recorder.NewHeartbeat(float64(heartbeatIntervalSecond), shutdownChan),
		id:                  idx,
		log:                 log,
		transactionRecorder: recorder.NewTransaction(),
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

//BroadcastReceiverClient - get zmq broadcast receiver remote
func (n *node) BroadcastReceiver() network.Client {
	return n.client.BroadcastReceiver()
}

//CommandSenderAndReceiver - network remote of command sender and receiver
func (n *node) CommandSenderAndReceiver() network.Client {
	return n.client.CommandSenderAndReceiver()
}

//CheckTimer - get sender timer
func (n *node) CheckTimer() *time.Timer {
	return n.checkTimer
}

//Remote - return remote interface
func (n *node) Client() Remote {
	return n.client
}

//CloseConnection - close connection
func (n *node) CloseConnection() error {
	if err := n.client.Close(); nil != err {
		return err
	}
	return nil
}

//DropRate - drop rate
func (n *node) DropRate() {
	return
}

//Log - get logger
func (n *node) Log() *logger.L {
	return n.log
}

//Monitor - start to monitor
func (n *node) Monitor() {
	rs := recorders{
		heartbeat:   n.heartbeatRecorder,
		transaction: n.transactionRecorder,
	}
	go receiverLoop(n, rs, n.id)
	go checkerLoop(n, rs)

	n.checkTimer.Reset(checkIntervalSecond)
	go senderLoop(n)

	<-shutdownChan

	n.Log().Info("stop")
	return
}

//StopMonitor - stop monitor
func (n *node) StopMonitor() {
	return
}

//Verify - verify record data
func (n *node) Verify() {
	return
}
