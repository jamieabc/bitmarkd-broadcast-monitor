package nodes

import (
	"context"
	"fmt"
	"sync"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/tasks"

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
	done    <-chan struct{}
	cancel  context.CancelFunc
	ctx     context.Context
	log     *logger.L
	nodeArr []node.Node
	tasks   tasks.Tasks
}

// Initialise - initialise objects
func Initialise(configs configuration.Configuration) (Nodes, error) {
	var ns []node.Node
	log := logger.New("nodes")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	t := tasks.NewTasks(done, cancel)

	nodeConfigs := configs.NodesConfig()
	node.Initialise(configs, t, ctx)
	t.Go(db.Start, ctx.Done())

	for idx, c := range nodeConfigs {
		n, err := node.NewNode(c, idx)
		if nil != err {
			return nil, err
		}
		ns = append(ns, n)
	}

	return &nodes{
		cancel:  cancel,
		ctx:     ctx,
		done:    done,
		log:     log,
		nodeArr: ns,
		tasks:   t,
	}, nil
}

// Monitor - start monitor
func (n *nodes) Monitor() {
	n.log.Info("start monitor")
	for _, connectedNode := range n.nodeArr {
		n.tasks.Go(connectedNode.Monitor)
	}

	<-n.ctx.Done()
	n.log.Info("receive stop signal")
	n.log.Flush()
}

// StopMonitor - stop monitor
func (n *nodes) StopMonitor() {
	n.log.Infof("stop monitor")
	fmt.Printf("stop\n")
	n.tasks.Done()
	n.log.Flush()
	<-n.done
}
