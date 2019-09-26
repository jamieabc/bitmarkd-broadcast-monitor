package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/db"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/fault"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"

	//_ "net/http/pprof"

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
	// pprof
	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()

	err := parseFlag()
	if nil != err {
		return
	}

	fmt.Printf("parse config...")
	config, err := configuration.Parse(configFile)
	if nil != err {
		fmt.Printf("parse config with error: %s", err)
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
		log.Errorf("authorize zmq with error: %s", err)
		return
	}

	log.Info("initialise db")
	err = db.Initialise(config.Influx(), logger.New("influxdb"))
	if nil != err {
		log.Errorf("initialise db with error: %s", err)
		return
	}

	log.Info("initialise nodes")
	n, err := nodes.Initialise(config)
	if nil != err {
		log.Errorf("initialise nodes with error: %s", err)
		return
	}

	log.Info("start monitor")
	go n.Monitor()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	fmt.Println("running...")
	<-ch
	fmt.Println("receive interrupt")
	n.StopMonitor()
	fmt.Println("finish stop monitor")
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
