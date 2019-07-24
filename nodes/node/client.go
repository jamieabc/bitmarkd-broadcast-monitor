package node

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/communication"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

type NodeClient interface {
	BroadcastReceiver() *network.Client
	Close()
	CommandSenderAndReceiver() *network.Client
	DigestOfHeight(height uint64) (*blockdigest.Digest, error)
	Info() (*communication.InfoResponse, error)
}

type NodeClientImpl struct {
	broadcastReceiver        *network.Client
	commandSenderAndReciever *network.Client
}

type connectionInfo struct {
	addressAndPort string
	chain          string
	zmqType        zmq.Type
}

func newClient(config configuration.NodeConfig, nodeKey *nodeKeys) (NodeClient, error) {
	broadcastReceiver, err := newZmqClient(nodeKey, connectionInfo{
		addressAndPort: broadcastAddressAndPort(config),
		chain:          config.Chain,
		zmqType:        zmq.SUB,
	})
	if nil != err {
		return nil, err
	}

	commandSenderAndReceiver, err := newZmqClient(nodeKey, connectionInfo{
		addressAndPort: commandAddressAndPort(config),
		chain:          config.Chain,
		zmqType:        zmq.REQ,
	})
	if nil != err {
		return nil, err
	}

	return &NodeClientImpl{
		broadcastReceiver:        broadcastReceiver,
		commandSenderAndReciever: commandSenderAndReceiver,
	}, nil
}

func newZmqClient(nodeKey *nodeKeys, info connectionInfo) (client *network.Client, err error) {
	address, err := network.NewConnection(info.addressAndPort)
	if nil != err {
		return nil, err
	}

	client, err = network.NewClient(info.zmqType, nodeKey.private, nodeKey.public, 0)
	if nil != err {
		return nil, err
	}
	defer func() {
		if nil != err && nil != client {
			network.CloseClients([]*network.Client{client})
		}
	}()

	err = client.Connect(address, nodeKey.remotePublic, info.chain)
	if nil != err {
		return nil, err
	}

	return client, nil
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

// BroadcastReceiver - zmq client of broadcast receiver
func (c *NodeClientImpl) BroadcastReceiver() *network.Client {
	return c.broadcastReceiver
}

func (n *NodeClientImpl) Close() {
	n.closeBroadcastReceiver()
	n.closeCommandSenderAndReceiver()
}

func (n *NodeClientImpl) closeBroadcastReceiver() {
	if nil != n.broadcastReceiver {
		n.broadcastReceiver.Close()
	}
}

func (n *NodeClientImpl) closeCommandSenderAndReceiver() {
	if nil != n.commandSenderAndReciever {
		n.commandSenderAndReciever.Close()
	}
}

// CommandSenderAndReceiver - zmq client of command sender and receiver
func (c *NodeClientImpl) CommandSenderAndReceiver() *network.Client {
	return c.commandSenderAndReciever
}

// DigestOfHeight - digest of block height
func (n *NodeClientImpl) DigestOfHeight(height uint64) (*blockdigest.Digest, error) {
	intf := communication.New(communication.ComDigest, n.commandSenderAndReciever)
	reply, err := intf.Get(height)
	if nil != err {
		return nil, err
	}
	return reply.(*blockdigest.Digest), nil
}

// Info - remote client info
func (n *NodeClientImpl) Info() (*communication.InfoResponse, error) {
	intf := communication.New(communication.ComInfo, n.commandSenderAndReciever)
	reply, err := intf.Get()
	if nil != err {
		return nil, err
	}

	return reply.(*communication.InfoResponse), nil
}
