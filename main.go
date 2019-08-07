package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/fault"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/nodes"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "c", "", "config file")
}

func main() {
	err := parseFlag()
	if nil != err {
		return
	}

	config, err := configuration.Parse(configFile)
	if nil != err {
		return
	}
	fmt.Printf("config: \n%s\n", config.String())

	err = initializeLogger(config)
	defer logger.Finalise()

	if nil != err {
		fmt.Printf("logger initialise with error: %s", err)
		return
	}

	log := logger.New("main")
	defer log.Info("shutdown...")

	log.Info("auth zmq")

	err = zmqAuth()
	if nil != err {
		return
	}

	log.Info("initialize nodes")

	n, err := initializeNodes(config)
	if nil != err {
		return
	}

	n.Monitor()

	fmt.Printf("sleep")
	time.Sleep(10 * time.Second)
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	fmt.Printf("hihi\n")
	n.StopMonitor()
	log.Flush()

	return
}

func parseFlag() error {
	flag.Parse()

	if "" == configFile {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", fault.InvalidEmptyConfigFile)
		return fault.InvalidEmptyConfigFile
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		_, _ = fmt.Fprintf(os.Stderr, "config file: %s, err: %s\n", configFile, fault.InvalidEmptyConfigFile)
		return fault.InvalidEmptyConfigFile
	}

	return nil
}

func initializeLogger(config configuration.Configuration) error {
	err := logger.Initialise(config.LogConfig())
	if nil != err {
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", err)
		return err
	}
	return nil
}

func zmqAuth() error {
	err := network.StartAuthentication()
	if nil != err {
		_, _ = fmt.Fprintf(os.Stderr, "zmq auth fail with error: %s\n", err)
		return err
	}
	return nil
}

func initializeNodes(config configuration.Configuration) (nodes.Nodes, error) {
	node, err := nodes.Initialise(config.NodesConfig(), config.Key(), config.HeartbeatIntervalInSecond())
	if nil != err {
		return nil, err
	}
	return node, nil
}
