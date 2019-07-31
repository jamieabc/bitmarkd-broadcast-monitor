package network

import (
	"github.com/pebbe/zmq4"
	zmq "github.com/pebbe/zmq4"
)

// SignalPair - zmq signal pair for receiving signal
type SignalPair interface {
	StopSender()
	StopReceiver()
	Send()
	Messages()
}

type signalPair struct {
	sender   *zmq4.Socket
	receiver *zmq.Socket
}

// NewSignalPair - new signal pair
func NewSignalPair(signal string) (receiver *zmq4.Socket, sender *zmq4.Socket, err error) {
	// PAIR server, half of signalling channel
	receiver, err = zmq4.NewSocket(zmq4.PAIR)
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
	sender, err = zmq4.NewSocket(zmq4.PAIR)
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
