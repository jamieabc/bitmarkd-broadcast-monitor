package node

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	zmq "github.com/pebbe/zmq4"
)

type NodeClient interface {
	BroadcastReceiver() *zmqutil.Client
	Close()
	CommandSenderAndReceiver() *zmqutil.Client
	DigestOfHeight(height uint64) (*blockdigest.Digest, error)
	Info() (*clientStatus, error)
}

type NodeClientImpl struct {
	broadcastReceiver        *zmqutil.Client
	commandSenderAndReciever *zmqutil.Client
}

type connectionInfo struct {
	addressAndPort string
	chain          string
	zmqType        zmq.Type
}

type clientStatus struct {
	Version string `json:"version"`
	Chain   string `json:"chain"`
	Normal  bool   `json:"normal"`
	Height  uint64 `json:"height"`
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

func newZmqClient(nodeKey *nodeKeys, info connectionInfo) (client *zmqutil.Client, err error) {
	address, err := util.NewConnection(info.addressAndPort)
	if nil != err {
		return nil, err
	}

	client, err = zmqutil.NewClient(info.zmqType, nodeKey.private, nodeKey.public, 0)
	if nil != err {
		return nil, err
	}
	defer func() {
		if nil != err && nil != client {
			zmqutil.CloseClients([]*zmqutil.Client{client})
		}
	}()

	err = client.Connect(address, nodeKey.remotePublic, info.chain)
	if nil != err {
		return nil, err
	}

	return client, nil
}

// BroadcastReceiver - zmq client of broadcast receiver
func (c *NodeClientImpl) BroadcastReceiver() *zmqutil.Client {
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
func (c *NodeClientImpl) CommandSenderAndReceiver() *zmqutil.Client {
	return c.commandSenderAndReciever
}

// DigestOfHeight - digest of block height
func (n *NodeClientImpl) DigestOfHeight(height uint64) (*blockdigest.Digest, error) {
	client := n.commandSenderAndReciever
	params := make([]byte, 8)
	binary.BigEndian.PutUint64(params, height)
	err := client.Send("H", params)
	if nil != err {
		return nil, err
	}

	data, err := client.Receive(0)
	if nil != err {
		return nil, err
	}
	if string(data[0]) != "H" || 2 != len(data) {
		return nil, fault.ErrInvalidPeerResponse
	}

	d := blockdigest.Digest{}
	err = blockdigest.DigestFromBytes(&d, data[1])
	return &d, err
}

// Info - remote client info
func (n *NodeClientImpl) Info() (*clientStatus, error) {
	client := n.commandSenderAndReciever
	err := client.Send("I")
	if nil != err {
		return nil, err
	}

	data, err := client.Receive(0)
	if nil != err {
		return nil, err
	}

	if "I" != string(data[0]) {
		return nil, fmt.Errorf("wrong command")
	}

	var info clientStatus
	err = json.Unmarshal(data[1], &info)
	if nil != err {
		return nil, err
	}
	fmt.Printf("info: %+v\n", info)

	return &info, nil
}
