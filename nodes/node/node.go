package node

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/messengers"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/recorder"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
)

// Node - node interface
type Node interface {
	BroadcastReceiver() network.Client
	Close() error
	CommandSender() network.Client
	Log() *logger.L
	Monitor()
	Name() string
	Remote() Remote
}

type recorders struct {
	heartbeat   recorder.Recorder
	transaction recorder.Recorder
	block       recorder.Recorder
}

type node struct {
	blockRecorder       recorder.Recorder
	config              configuration.NodeConfig
	heartbeatRecorder   recorder.Recorder
	id                  int
	log                 *logger.L
	name                string
	remote              Remote
	transactionRecorder recorder.Recorder
}

type nodeKeys struct {
	private      []byte
	public       []byte
	remotePublic []byte
}

var (
	shutdownChan            <-chan struct{}
	heartbeatIntervalSecond int
	keys                    configuration.Keys
	slack                   messengers.Messenger
)

// Initialise - setup node related common variables
func Initialise(shutdown <-chan struct{}, configs configuration.Configuration) {
	shutdownChan = shutdown
	recorder.Initialise(shutdown)
	heartbeatIntervalSecond = configs.HeartbeatIntervalInSecond()
	keys = configs.Key()

	slackConfig := configs.SlackConfig()
	slack = messengers.NewSlack(slackConfig.Token, slackConfig.ChannelID)
}

// NewNode - create new node
func NewNode(config configuration.NodeConfig, idx int) (intf Node, err error) {
	log := logger.New(config.Name)

	n := &node{
		blockRecorder:       recorder.NewBlock(),
		config:              config,
		heartbeatRecorder:   recorder.NewHeartbeat(float64(heartbeatIntervalSecond), shutdownChan),
		id:                  idx,
		log:                 log,
		name:                config.Name,
		transactionRecorder: recorder.NewTransaction(),
	}

	nodeKey, err := parseKeys(keys, config.PublicKey)
	if nil != err {
		log.Errorf("parse keys: %v, remote public key: %q with error: %s", keys, config.PublicKey, err)
		return nil, err
	}

	log.Debugf("public key: %q, private key: %q, remote public key: %q", nodeKey.public, nodeKey.private, nodeKey.remotePublic)

	n.remote, err = newClient(config, nodeKey)
	if nil != err {
		return nil, err
	}
	log.Infof("new node: %s", n.Name())

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
	return n.remote.BroadcastReceiver()
}

// CommandSender - network remote of command sender and receiver
func (n *node) CommandSender() network.Client {
	return n.remote.CommandSender()
}

// Remote - return remote interface
func (n *node) Remote() Remote {
	return n.remote
}

// Close - close connection
func (n *node) Close() error {
	if err := n.remote.Close(); nil != err {
		return err
	}
	return nil
}

// Log - get logger
func (n *node) Log() *logger.L {
	return n.log
}

// Monitor - start to monitor
func (n *node) Monitor() {
	rs := recorders{
		heartbeat:   n.heartbeatRecorder,
		transaction: n.transactionRecorder,
		block:       n.blockRecorder,
	}

	n.log.Info("start to monitor")
	go receiverLoop(n, rs, n.id)
	go checkerLoop(n, rs)

	if n.config.CommandPort != "" {
		go senderLoop(n)
	}

	<-shutdownChan

	n.Log().Info("stop")
	return
}

// Name - return node name
func (n *node) Name() string {
	return n.name
}
