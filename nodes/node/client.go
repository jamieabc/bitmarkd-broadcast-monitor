package node

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/communication"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

// Client - client interface
type Client interface {
	BroadcastReceiver() *network.Client
	Close() error
	CommandSenderAndReceiver() *network.Client
	DigestOfHeight(height uint64) (*blockdigest.Digest, error)
	Info() (*communication.InfoResponse, error)
}

type client struct {
	broadcastReceiver        *network.Client
	commandSenderAndReceiver *network.Client
}

type connectionInfo struct {
	addressAndPort string
	chain          string
	zmqType        zmq.Type
}

func newClient(config configuration.NodeConfig, nodeKey *nodeKeys) (Client, error) {
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

	return &client{
		broadcastReceiver:        broadcastReceiver,
		commandSenderAndReceiver: commandSenderAndReceiver,
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
func (c *client) BroadcastReceiver() *network.Client {
	return c.broadcastReceiver
}

// Close - close client
func (c *client) Close() error {
	if err := c.closeBroadcastReceiver(); nil != err {
		return err
	}
	if err := c.closeCommandSenderAndReceiver(); nil != err {
		return err
	}
	return nil
}

func (c *client) closeBroadcastReceiver() error {
	if nil != c.broadcastReceiver {
		if err := c.broadcastReceiver.Close(); nil != err {
			return err
		}
	}
	return nil
}

func (c *client) closeCommandSenderAndReceiver() error {
	if nil != c.commandSenderAndReceiver {
		if err := c.commandSenderAndReceiver.Close(); nil != err {
			return err
		}
	}
	return nil
}

// CommandSenderAndReceiver - zmq client of command sender and receiver
func (c *client) CommandSenderAndReceiver() *network.Client {
	return c.commandSenderAndReceiver
}

// DigestOfHeight - digest of block height
func (c *client) DigestOfHeight(height uint64) (*blockdigest.Digest, error) {
	comm := communication.New(communication.ComDigest, c.commandSenderAndReceiver)
	reply, err := comm.Get(height)
	if nil != err {
		return nil, err
	}
	return reply.(*blockdigest.Digest), nil
}

// Info - remote client info
func (c *client) Info() (*communication.InfoResponse, error) {
	comm := communication.New(communication.ComInfo, c.commandSenderAndReceiver)
	reply, err := comm.Get()
	if nil != err {
		return nil, err
	}

	return reply.(*communication.InfoResponse), nil
}
