package node

import (
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	zmq "github.com/pebbe/zmq4"
)

type NodeClient interface {
	BroadcastReceiver() *zmqutil.Client
	CloseBroadcastReceiver()
	CloseCommandSenderAndReceiver()
	CommandSenderAndReceiver() *zmqutil.Client
}

type NodeClientImpl struct {
	broadcastReceiver        *zmqutil.Client
	commandSenderAndReciever *zmqutil.Client
}

func newClient(config configuration.NodeConfig, nodeKey *nodeKeys) (NodeClient, error) {
	broadcastReceiver, err := newZmqClient(broadcastAddressAndPort(config), nodeKey, config.Chain)
	if nil != err {
		return nil, err
	}

	commandSenderAndReceiver, err := newZmqClient(commandAddressAndPort(config), nodeKey, config.Chain)
	if nil != err {
		return nil, err
	}

	return &NodeClientImpl{
		broadcastReceiver:        broadcastReceiver,
		commandSenderAndReciever: commandSenderAndReceiver,
	}, nil
}

func newZmqClient(addressAndPort string, nodeKey *nodeKeys, chain string) (client *zmqutil.Client, err error) {
	address, err := util.NewConnection(addressAndPort)
	if nil != err {
		return nil, err
	}

	client, err = zmqutil.NewClient(zmq.SUB, nodeKey.private, nodeKey.public, 0)
	if nil != err {
		return nil, err
	}
	defer func() {
		if nil != err && nil != client {
			zmqutil.CloseClients([]*zmqutil.Client{client})
		}
	}()

	err = client.Connect(address, nodeKey.remotePublic, chain)
	if nil != err {
		return nil, err
	}

	return client, nil
}

// BroadcastReceiver - zmq client of broadcast receiver
func (c *NodeClientImpl) BroadcastReceiver() *zmqutil.Client {
	return c.broadcastReceiver
}

// CloseBroadcastReceiver - close broadcast receiver client
func (n *NodeClientImpl) CloseBroadcastReceiver() {
	if nil != n.broadcastReceiver {
		n.broadcastReceiver.Close()
	}
}

// CloseCommandSenderAndReceiver - close command sender and receiver client
func (n *NodeClientImpl) CloseCommandSenderAndReceiver() {
	if nil != n.commandSenderAndReciever {
		n.commandSenderAndReciever.Close()
	}
}

// CommandSenderAndReceiver - zmq client of command sender and receiver
func (c *NodeClientImpl) CommandSenderAndReceiver() *zmqutil.Client {
	return c.commandSenderAndReciever
}
