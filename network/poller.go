package network

import (
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/logger"

	zmq "github.com/pebbe/zmq4"
)

// structure to hold a poll
type poller struct {
	sync.Mutex
	eventChan    chan zmq.Polled
	signalPair   signalPairer
	poll         *zmq.Poller
	sockets      map[*zmq.Socket]zmq.State
	shutdownChan <-chan struct{}
}

const (
	signalFormat = "inproc://monitor-poll-internal-signal-%d"
)

// NewPoller - create poller
// this is just to encapsulate the zmq poll to allow removal of a socket from a socket
func NewPoller(eventChannel chan zmq.Polled, shutdownChannel <-chan struct{}, id int) (Poller, error) {
	var err error

	signalString := fmt.Sprintf(signalFormat, id)
	signalPair, err := newSignalPair(signalString)
	if nil != err {
		return nil, err
	}

	poll := zmq.NewPoller()
	poll.Add(signalPair.Receiver(), zmq.POLLIN)

	return &poller{
		eventChan:    eventChannel,
		poll:         poll,
		signalPair:   signalPair,
		shutdownChan: shutdownChannel,
		sockets:      make(map[*zmq.Socket]zmq.State),
	}, nil
}

// Add - add socket to poll
func (p *poller) Add(client Client, events zmq.State) {
	p.Lock()
	defer p.Unlock()

	socket := client.Socket()
	fmt.Printf("socket: %+v\n", socket)

	// protect against duplicate add
	if _, exist := p.sockets[socket]; exist {
		fmt.Printf("socket %+v already exist", socket)
		return
	}

	// preserve the event mask
	p.sockets[socket] = events

	// add to the internal p
	p.poll.Add(socket, events)
}

// remove a socket from a poll
func (p *poller) Remove(client Client) {
	socket := client.Socket()
	p.Lock()
	defer p.Unlock()

	// protect against duplicate remove
	if _, ok := p.sockets[socket]; !ok {
		return
	}

	// remove the socket
	delete(p.sockets, socket)
	_ = p.poll.RemoveBySocket(socket)
}

// Start - polling event
func (p *poller) Start(timeout time.Duration) {
	go waitShutdownEvent(p)

	for {
		p.Lock()
		poll := p.poll
		p.Unlock()

		polled, _ := poll.Poll(timeout)

		for _, zmqEvent := range polled {
			if p.signalPair.Receiver() == zmqEvent.Socket {
				logger.Critical("receive internal signal pair, terminate")
				return
			}
			p.eventChan <- zmqEvent
		}
	}
}

func (p *poller) stop() {
	p.signalPair.Stop()
}

func waitShutdownEvent(p *poller) {
	<-p.shutdownChan
	p.stop()
}

// Poller - poll interface
type Poller interface {
	Add(client Client, events zmq.State)
	Remove(client Client)
	Start(timeout time.Duration)
}
