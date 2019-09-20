package nodes

import (
	"fmt"
	"sync"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/messengers"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/nodes/node"
)

// Nodes - nodes interface
type Nodes interface {
	Monitor()
	StopMonitor()
}

type nodes struct {
	sync.RWMutex
	log          *logger.L
	nodeArr      []node.Node
	shutdownChan chan struct{}
	slack        messengers.Messenger
}

// Initialise - initialise objects
func Initialise(configs configuration.Configuration) (Nodes, error) {
	var ns []node.Node
	log := logger.New("nodes")
	shutdownCh := make(chan struct{})

	nodeConfigs := configs.NodesConfig()
	node.Initialise(shutdownCh, configs)
	go db.Start(shutdownCh)

	for idx, c := range nodeConfigs {
		n, err := node.NewNode(c, idx)
		if nil != err {
			return nil, err
		}
		ns = append(ns, n)
	}

	slackConfig := configs.SlackConfig()

	return &nodes{
		log:          log,
		nodeArr:      ns,
		shutdownChan: shutdownCh,
		slack:        messengers.NewSlack(slackConfig.Token, slackConfig.ChannelID),
	}, nil
}

// Monitor - start monitor
func (n *nodes) Monitor() {
	n.log.Info("start monitor")
	for _, connectedNode := range n.nodeArr {
		go connectedNode.Monitor()
	}

	<-n.shutdownChan
	n.log.Info("receive stop signal")
	n.log.Flush()
}

// StopMonitor - stop monitor
func (n *nodes) StopMonitor() {
	n.log.Infof("stop monitor")
	fmt.Printf("stop\n")
	close(n.shutdownChan)
	n.log.Flush()
}
