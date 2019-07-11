package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/nodes"
)

const (
	errEmptyConfigFile = "empty config file"
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

	log := logger.New("main")
	defer log.Info("shutdown...")

	if nil != err {
		return
	}

	log.Info("auth zmq")

	err = zmqAuth()
	if nil != err {
		return
	}

	log.Info("initialize nodes")

	node, err := initializeNodes(config)
	if nil != err {
		return
	}
	defer node.StopMonitor()

	node.Monitor()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Infof("receive signal: %v", sig)

	return
}

func parseFlag() error {
	flag.Parse()

	if "" == configFile {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "\n%s\n", errEmptyConfigFile)
		return errors.New(errEmptyConfigFile)
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "config file: %s, err: %s\n", configFile, errEmptyConfigFile)
		return errors.New(errEmptyConfigFile)
	}

	return nil
}

func initializeLogger(config configuration.Configuration) error {
	err := logger.Initialise(config.LogConfig())
	if nil != err {
		fmt.Fprintf(os.Stderr, "\n%s\n", err)
		return err
	}
	return nil
}

func zmqAuth() error {
	err := zmqutil.StartAuthentication()
	if nil != err {
		fmt.Fprintf(os.Stderr, "zmq auth fail with error: %s\n", err)
		return err
	}
	return nil
}

func initializeNodes(config configuration.Configuration) (nodes.Nodes, error) {
	node, err := nodes.Initialise(config.NodesConfig(), config.Key())
	if nil != err {
		return nil, err
	}
	return node, nil
}
