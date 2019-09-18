package node

import (
	"fmt"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/communication"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
	zmq "github.com/pebbe/zmq4"
)

// Remote - remote interface
type Remote interface {
	BlockHeader(uint64) (*communication.BlockHeaderResponse, error)
	BroadcastReceiver() network.Client
	Close() error
	CommandSender() network.Client
	Info() (*communication.InfoResponse, error)
	Height() (*communication.HeightResponse, error)
}

type remote struct {
	broadcastReceiver network.Client
	commandSender     network.Client
}

type connectionInfo struct {
	addressAndPort string
	chain          string
	zmqType        zmq.Type
}

func newClient(config configuration.NodeConfig, nodeKey *nodeKeys) (Remote, error) {
	broadcastReceiver, err := newZmqClient(nodeKey, connectionInfo{
		addressAndPort: broadcastAddressAndPort(config),
		chain:          config.Chain,
		zmqType:        zmq.SUB,
	})
	if nil != err {
		return nil, err
	}

	var commandSenderAndReceiver network.Client

	if config.CommandPort != "" {
		commandSenderAndReceiver, err = newZmqClient(nodeKey, connectionInfo{
			addressAndPort: commandAddressAndPort(config),
			chain:          config.Chain,
			zmqType:        zmq.REQ,
		})
		if nil != err {
			return nil, err
		}
	}

	return &remote{
		broadcastReceiver: broadcastReceiver,
		commandSender:     commandSenderAndReceiver,
	}, nil
}

func newZmqClient(nodeKey *nodeKeys, info connectionInfo) (client network.Client, err error) {
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
			network.CloseClients([]network.Client{client})
		}
	}()

	err = client.Connect(address, nodeKey.remotePublic, info.chain)
	if nil != err {
		return nil, err
	}

	return client, nil
}

func broadcastAddressAndPort(config configuration.NodeConfig) string {
	return hostAndPort(config.IP, config.BroadcastPort)
}

func commandAddressAndPort(config configuration.NodeConfig) string {
	return hostAndPort(config.IP, config.CommandPort)
}

func hostAndPort(host string, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

//BroadcastReceiver - zmq remote of broadcast receiver
func (r *remote) BroadcastReceiver() network.Client {
	return r.broadcastReceiver
}

//Close - close remote
func (r *remote) Close() error {
	if err := r.closeBroadcastReceiver(); nil != err {
		return err
	}
	if err := r.closeCommandSenderAndReceiver(); nil != err {
		return err
	}
	return nil
}

func (r *remote) closeBroadcastReceiver() error {
	if nil != r.broadcastReceiver {
		if err := r.broadcastReceiver.Close(); nil != err {
			return err
		}
	}
	return nil
}

func (r *remote) closeCommandSenderAndReceiver() error {
	if nil != r.commandSender {
		if err := r.commandSender.Close(); nil != err {
			return err
		}
	}
	return nil
}

// CommandSender - zmq remote of command sender and receiver
func (r *remote) CommandSender() network.Client {
	return r.commandSender
}

// Info - remote info
func (r *remote) Info() (*communication.InfoResponse, error) {
	comm := communication.New(communication.ComInfo, r.commandSender)
	reply, err := comm.Get()
	if nil != err {
		return nil, err
	}

	return reply.(*communication.InfoResponse), nil
}

// Height - remote height
func (r *remote) Height() (*communication.HeightResponse, error) {
	comm := communication.New(communication.ComHeight, r.commandSender)
	reply, err := comm.Get()
	if nil != err {
		return nil, err
	}
	return reply.(*communication.HeightResponse), nil
}

// BlockHeader - block header
func (r *remote) BlockHeader(height uint64) (*communication.BlockHeaderResponse, error) {
	comm := communication.New(communication.ComBlockHeader, r.commandSender)
	reply, err := comm.Get(height)
	if nil != err {
		return nil, err
	}
	return reply.(*communication.BlockHeaderResponse), nil
}
