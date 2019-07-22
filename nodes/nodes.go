package nodes

import (
	"sync"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/nodes/node"
)

type Nodes interface {
	DropRate()
	Monitor()
	StopMonitor()
}

type NodesImpl struct {
	sync.RWMutex
	log      *logger.L
	nodes    []node.Node
	shutdown chan struct{}
}

// Initialise - initialise nodes
func Initialise(configs []configuration.NodeConfig, keys configuration.Keys) (Nodes, error) {
	nodes := []node.Node{}
	log := logger.New("nodes")

	err := node.Initialise()
	if nil != err {
		log.Errorf("node initialise with error: %s", err)
		return nil, err
	}

	for idx, c := range configs {
		n, err := node.NewNode(c, keys, idx)
		if nil != err {
			return nil, err
		}
		nodes = append(nodes, n)
	}

	return &NodesImpl{
		log:      log,
		nodes:    nodes,
		shutdown: make(chan struct{}),
	}, nil
}

// DropRate - drop rate
func (n *NodesImpl) DropRate() {
}

// Monitor - start monitor
func (n *NodesImpl) Monitor() {
	nodeShutdown := make(chan struct{})
	n.log.Info("start monitor")
	for _, connectedNode := range n.nodes {
		go connectedNode.Monitor(nodeShutdown)
	}

	select {
	case <-n.shutdown:
		nodeShutdown <- struct{}{}
		n.log.Info("receive stop signal")
		break
	}

	defer func() {
		n.log.Flush()
	}()
}

// StopMonitor - stop monitor
func (n *NodesImpl) StopMonitor() {
	n.log.Infof("stop monitor")
	n.shutdown <- struct{}{}
	n.log.Flush()
}
