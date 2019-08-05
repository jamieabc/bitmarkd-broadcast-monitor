package nodes

import (
	"sync"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/nodes/node"
)

//Nodes - nodes interface
type Nodes interface {
	DropRate()
	Monitor()
	StopMonitor()
}

type nodes struct {
	sync.RWMutex
	log      *logger.L
	nodeArr  []node.Node
	shutdown chan struct{}
}

//Initialise - initialise nodes
func Initialise(configs []configuration.NodeConfig, keys configuration.Keys) (Nodes, error) {
	var ns []node.Node
	log := logger.New("nodes")
	shutdownChan := make(chan struct{})
	node.Initialise(shutdownChan)

	for idx, c := range configs {
		n, err := node.NewNode(c, keys, idx)
		if nil != err {
			return nil, err
		}
		ns = append(ns, n)
	}

	return &nodes{
		log:      log,
		nodeArr:  ns,
		shutdown: shutdownChan,
	}, nil
}

//DropRate - drop rate
func (n *nodes) DropRate() {
}

//Monitor - start monitor
func (n *nodes) Monitor() {
	n.log.Info("start monitor")
	for _, connectedNode := range n.nodeArr {
		go connectedNode.Monitor()
	}

	select {
	case <-n.shutdown:
		n.shutdown <- struct{}{}
		n.log.Info("receive stop signal")
		break
	}

	defer func() {
		n.log.Flush()
	}()
}

//StopMonitor - stop monitor
func (n *nodes) StopMonitor() {
	n.log.Infof("stop monitor")
	n.shutdown <- struct{}{}
	n.log.Flush()
}
