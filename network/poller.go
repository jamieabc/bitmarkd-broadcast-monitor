package network

import (
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
)

// structure to hold a poller
type poller struct {
	sync.Mutex
	sockets map[*zmq.Socket]zmq.State
	poller  *zmq.Poller
}

// create a poller
// this is just to encapsulate the zmq poller to allow removal of a socket from a socket
func NewPoller() Poller {
	return &poller{
		sockets: make(map[*zmq.Socket]zmq.State),
		poller:  zmq.NewPoller(),
	}
}

// Add - add socket to poller
func (p *poller) Add(client Client, events zmq.State) {
	p.Lock()
	defer p.Unlock()

	socket := client.Socket()

	// protect against duplicate add
	if _, exist := p.sockets[socket]; exist {
		return
	}

	// preserve the event mask
	p.sockets[socket] = events

	// add to the internal p
	p.poller.Add(socket, events)
}

// remove a socket from a poller
func (p *poller) Remove(socket *zmq.Socket) {

	p.Lock()
	defer p.Unlock()

	// protect against duplicate remove
	if _, ok := p.sockets[socket]; !ok {
		return
	}

	// remove the socket
	delete(p.sockets, socket)
	_ = p.poller.RemoveBySocket(socket)
}

// perform a poll
func (p *poller) Poll(timeout time.Duration) ([]zmq.Polled, error) {
	p.Lock()
	poll := p.poller
	p.Unlock()
	polled, err := poll.Poll(timeout)
	return polled, err
}

// Poller - poller interface
type Poller interface {
	Add(client Client, events zmq.State)
	Poll(timeout time.Duration) ([]zmq.Polled, error)
	Remove(socket *zmq.Socket)
}
