package network

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// point at which to disconnect large message senders
// current estimate of a block maximum is 2 MB
const (
	maximumPacketSize = 5000000 // 5 MB
)

// structure to hold a client connection
//
// prefix:
//   REQ socket this adds an item before send
//   SUB socket this adds/changes subscription
type client struct {
	sync.Mutex

	publicKey       []byte
	privateKey      []byte
	serverPublicKey []byte
	address         string
	prefix          string
	v6              bool
	socketType      zmq.Type
	socket          *zmq.Socket
	events          zmq.State
	timeout         time.Duration
	timestamp       time.Time
}

const (
	publicKeySize  = 32
	privateKeySize = 32
	identifierSize = 32
)

// create a client socket usually of type zmq.REQ or zmq.SUB
func NewClient(socketType zmq.Type, privateKey []byte, publicKey []byte, timeout time.Duration) (*client, error) {

	if len(publicKey) != publicKeySize {
		return nil, fault.ErrInvalidPublicKey
	}
	if len(privateKey) != privateKeySize {
		return nil, fault.ErrInvalidPrivateKey
	}

	client := &client{
		publicKey:       make([]byte, publicKeySize),
		privateKey:      make([]byte, privateKeySize),
		serverPublicKey: make([]byte, publicKeySize),
		address:         "",
		v6:              false,
		socketType:      socketType,
		socket:          nil,
		events:          0,
		timeout:         timeout,
		timestamp:       time.Now(),
	}
	copy(client.privateKey, privateKey)
	copy(client.publicKey, publicKey)
	return client, nil
}

// create a socket and connect to specific server with specified key
func (client *client) openSocket() error {
	client.Lock()
	defer client.Unlock()

	socket, err := zmq.NewSocket(client.socketType)
	if nil != err {
		return err
	}

	// create a secure random identifier
	randomIdBytes := make([]byte, identifierSize)
	_, err = rand.Read(randomIdBytes)
	if nil != err {
		return err
	}
	randomIdentifier := string(randomIdBytes)

	// set up as client
	err = socket.SetCurveServer(0)
	if nil != err {
		goto failure
	}
	err = socket.SetCurvePublickey(string(client.publicKey))
	if nil != err {
		goto failure
	}
	err = socket.SetCurveSecretkey(string(client.privateKey))
	if nil != err {
		goto failure
	}

	// local identity is a random value
	err = socket.SetIdentity(randomIdentifier)
	if nil != err {
		goto failure
	}

	// destination identity is its public key
	err = socket.SetCurveServerkey(string(client.serverPublicKey))
	if nil != err {
		goto failure
	}

	// only queue messages sent to connected peers
	_ = socket.SetImmediate(true)

	// zero => do not set timeout
	if 0 != client.timeout {
		err = socket.SetSndtimeo(client.timeout)
		if nil != err {
			goto failure
		}
		err = socket.SetRcvtimeo(client.timeout)
		if nil != err {
			goto failure
		}
	}
	err = socket.SetLinger(100 * time.Millisecond)
	if nil != err {
		goto failure
	}

	// socket type specific options
	switch client.socketType {
	case zmq.REQ:
		err = socket.SetReqCorrelate(1)
		if nil != err {
			goto failure
		}
		err = socket.SetReqRelaxed(1)
		if nil != err {
			goto failure
		}

	case zmq.SUB:
		// set subscription prefix - empty => receive everything
		err = socket.SetSubscribe(client.prefix)
		if nil != err {
			goto failure
		}

	default:
	}

	err = socket.SetTcpKeepalive(1)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveCnt(5)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveIdle(60)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveIntvl(60)
	if nil != err {
		goto failure
	}

	// see socket.go for constants
	err = socket.SetMaxmsgsize(maximumPacketSize)
	if nil != err {
		goto failure
	}

	// set IPv6 state before connect
	err = socket.SetIpv6(client.v6)
	if nil != err {
		goto failure
	}

	// new connection
	err = socket.Connect(client.address)
	if nil != err {
		goto failure
	}

	client.socket = socket

	return nil
failure:
	socket.Close()
	return err
}

// destroy the socket, but leave other connection info so can reconnect
// to the same endpoint again
func (client *client) closeSocket() error {
	client.Lock()
	defer client.Unlock()

	if nil == client.socket {
		return nil
	}

	// if already connected, disconnect first
	if "" != client.address {
		_ = client.socket.Disconnect(client.address)
	}

	// close socket
	err := client.socket.Close()
	client.socket = nil
	return err
}

// disconnect old address and connect to new
func (client *client) Connect(conn *Connection, serverPublicKey []byte, prefix string) error {
	// if already connected, disconnect first
	err := client.closeSocket()
	if nil != err {
		return err
	}
	client.address = ""
	client.prefix = prefix

	// small delay to allow any background socket closing
	// and to restrict rate of reconnection
	time.Sleep(5 * time.Millisecond)

	copy(client.serverPublicKey, serverPublicKey)

	client.address, client.v6 = conn.canonicalIPandPort("tcp://")

	client.timestamp = time.Now()

	return client.openSocket()
}

// check if connected to a node
func (client *client) IsConnected() bool {
	return "" != client.address
}

// check if connected to a specific node
func (client *client) IsConnectedTo(serverPublicKey []byte) bool {
	return bytes.Equal(client.serverPublicKey, serverPublicKey)
}

// close and reopen the connection
func (client *client) Reconnect() error {
	_, err := client.reopenSocket()
	return err
}

// close and open socket
func (client *client) reopenSocket() (*zmq.Socket, error) {
	err := client.closeSocket()
	if nil != err {
		return nil, err
	}
	<-time.After(20 * time.Millisecond)
	err = client.openSocket()
	if nil != err {
		return nil, err
	}
	return client.socket, nil
}

// disconnect old address and close
func (client *client) Close() error {
	return client.closeSocket()
}

// disconnect old addresses and close all
func CloseClients(clients []Client) {
	for _, client := range clients {
		if nil != client {
			_ = client.Close()
		}
	}
}

// send a message
func (client *client) Send(items ...interface{}) error {
	client.Lock()
	defer client.Unlock()

	if "" == client.address {
		return fault.ErrNotConnected
	}

	if 0 == len(items) {
		logger.Panicf("client.Send no arguments provided")
	}

	if "" != client.prefix {
		_, err := client.socket.Send(client.prefix, zmq.SNDMORE)
		if nil != err {
			return err
		}
	}

	n := len(items) - 1
	a := items[:n]
	final := items[n] // just the final item

	for i, item := range a {
		switch it := item.(type) {
		case string:
			_, err := client.socket.Send(it, zmq.SNDMORE)
			if nil != err {
				return err
			}
		case []byte:
			_, err := client.socket.SendBytes(it, zmq.SNDMORE)
			if nil != err {
				return err
			}
		case [][]byte:
			for _, sub := range it {
				_, err := client.socket.SendBytes(sub, zmq.SNDMORE)
				if nil != err {
					return err
				}
			}
		default:
			logger.Panicf("client.Send cannot send[%d]: %#v", i, item)
		}
	}

	switch it := final.(type) {
	case string:
		_, err := client.socket.Send(it, 0)
		if nil != err {
			return err
		}
	case []byte:
		_, err := client.socket.SendBytes(it, 0)
		if nil != err {
			return err
		}
	case [][]byte:
		if 0 == len(it) {
			logger.Panicf("client.Send empty [][]byte")
		}
		n := len(it) - 1
		a := it[:n]
		last := it[n] // just the final item []byte

		for _, sub := range a {
			_, err := client.socket.SendBytes(sub, zmq.SNDMORE)
			if nil != err {
				return err
			}
		}
		_, err := client.socket.SendBytes(last, 0)
		if nil != err {
			return err
		}

	default:
		logger.Panicf("client.Send cannot send[%d]: %#v", n, final)
	}

	return nil
}

// receive a reply
func (client *client) Receive(flags zmq.Flag) ([][]byte, error) {
	client.Lock()
	defer client.Unlock()

	if "" == client.address {
		return nil, fault.ErrNotConnected
	}
	data, err := client.socket.RecvMessageBytes(flags)
	return data, err
}

// to string
func (client *client) String() string {
	return client.address
}

type connected struct {
	Address string `json:"address"`
	Server  string `json:"server"`
}

// ConnectedTo - return connected remote client info
func (client *client) ConnectedTo() *connected {

	if "" == client.address {
		return nil
	}
	return &connected{
		Address: client.address,
		Server:  hex.EncodeToString(client.serverPublicKey),
	}
}

// Socket - return socket
func (client *client) Socket() *zmq.Socket {
	return client.socket
}

// Remote - network client interface
type Client interface {
	Close() error
	Connect(conn *Connection, serverPublicKey []byte, prefix string) error
	ConnectedTo() *connected
	IsConnected() bool
	IsConnectedTo(serverPublicKey []byte) bool
	Receive(flags zmq.Flag) ([][]byte, error)
	Reconnect() error
	Send(items ...interface{}) error
	Socket() *zmq.Socket
	String() string
}
