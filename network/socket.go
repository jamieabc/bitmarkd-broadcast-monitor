package network

import (
	zmq "github.com/pebbe/zmq4"
)

// point at which to disconnect large message senders
// current estimate of a block maximum is 2 MB
const (
	maximumPacketSize = 5000000 // 5 MB
)

// ***** FIX THIS: enabling this causes complete failure
// ***** FIX THIS: socket disconnects, perhaps after IVL value
// const (
// 	heartbeatInterval = 15 * time.Second
// 	heartbeatTimeout  = 60 * time.Second
// 	heartbeatTTL      = 60 * time.Second
// )

// return a pair of connected PAIR sockets
// for shutdown signalling
func NewSignalPair(signal string) (reciever *zmq.Socket, sender *zmq.Socket, err error) {

	// PAIR server, half of signalling channel
	reciever, err = zmq.NewSocket(zmq.PAIR)
	if nil != err {
		return nil, nil, err
	}
	_ = reciever.SetLinger(0)
	err = reciever.Bind(signal)
	if nil != err {
		_ = reciever.Close()
		return nil, nil, err
	}

	// PAIR Client, half of signalling channel
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

	return reciever, sender, nil
}
