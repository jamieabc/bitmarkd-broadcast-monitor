package network

import (
	"github.com/pebbe/zmq4"
	zmq "github.com/pebbe/zmq4"
)

// signalPairer - zmq signal pair for receiving signal
type signalPairer interface {
	ReceiverChan() <-chan string
	Receiver() *zmq.Socket
	Start()
	Stop()
	Send(interface{}) error
}

type signalPair struct {
	sender          *zmq4.Socket
	receiver        *zmq.Socket
	receiverChannel chan string
}

const (
	chanSize = 10
	stopMsg  = "stop"
)

// ReceiverChan - return a channel for received message
func (s *signalPair) ReceiverChan() <-chan string {
	return s.receiverChannel
}

// Receiver - return receiver socket
func (s *signalPair) Receiver() *zmq.Socket {
	return s.receiver
}

// Start - start signal pair to work
func (s *signalPair) Start() {
	shutdownChan := make(chan struct{})
	go s.receiverLoop(nil)
	<-shutdownChan
}

func (s *signalPair) receiverLoop(shutdownChan chan struct{}) {
	for {
		str, err := s.receiver.Recv(0)
		if nil != err {
			continue
		}
		s.receiverChannel <- str
		if str == stopMsg {
			shutdownChan <- struct{}{}
			_ = s.stopReceiver()
			return
		}
	}
}

// Stop - stop signal pair
func (s *signalPair) Stop() {
	_ = s.stopSender()
}

func (s *signalPair) stopSender() error {
	_ = s.Send(stopMsg)
	if err := s.sender.Close(); nil != err {
		return err
	}
	return nil
}

func (s *signalPair) stopReceiver() error {
	if err := s.receiver.Close(); nil != err {
		return err
	}
	return nil
}

// Send - send message
func (s *signalPair) Send(msg interface{}) error {
	if _, err := s.sender.SendMessage(msg.(string)); nil != err {
		return err
	}
	return nil
}

// newSignalPair - new signal pair
func newSignalPair(signal string) (signalPairer, error) {
	// PAIR server, half of signalling channel
	receiver, err := zmq4.NewSocket(zmq4.PAIR)
	if nil != err {
		return nil, err
	}

	_ = receiver.SetLinger(0)
	err = receiver.Bind(signal)
	if nil != err {
		closeSocket(receiver)
		return nil, err
	}

	// PAIR client, half of signalling channel
	sender, err := zmq4.NewSocket(zmq4.PAIR)
	if nil != err {
		closeSocket(sender)
		return nil, err
	}

	_ = sender.SetLinger(0)
	err = sender.Connect(signal)
	if nil != err {
		closeSocket(sender)
		return nil, err
	}

	ch := make(chan string, chanSize)

	return &signalPair{sender, receiver, ch}, nil
}

func closeSocket(socket *zmq.Socket) {
	if nil != socket {
		_ = socket.Close()
	}
}
