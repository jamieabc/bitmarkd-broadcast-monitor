package network

import (
	zmq "github.com/pebbe/zmq4"
)

// point at which to disconnect large message senders
// current estimate of a block maximum is 2 MB
const (
	maximumPacketSize = 5000000 // 5 MB
)

// return a pair of connected PAIR sockets
// for shutdown signalling
func NewSignalPair(signal string) (receiver *zmq.Socket, sender *zmq.Socket, err error) {

	// PAIR server, half of signalling channel
	receiver, err = zmq.NewSocket(zmq.PAIR)
	if nil != err {
		return nil, nil, err
	}
	_ = receiver.SetLinger(0)
	err = receiver.Bind(signal)
	if nil != err {
		_ = receiver.Close()
		return nil, nil, err
	}

	// PAIR client, half of signalling channel
	sender, err = zmq.NewSocket(zmq.PAIR)
	if nil != err && nil != sender {
		_ = sender.Close()
		return nil, nil, err
	}
	_ = sender.SetLinger(0)
	err = sender.Connect(signal)
	if nil != err {
		_ = sender.Close()
		return nil, nil, err
	}

	return receiver, sender, nil
}
