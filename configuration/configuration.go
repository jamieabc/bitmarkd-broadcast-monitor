package configuration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitmark-inc/logger"
)

//Configuration - configuration interface
type Configuration interface {
	Data() *configuration
	HeartbeatIntervalInSecond() int
	Influx() InfluxDBConfig
	Key() Keys
	LogConfig() logger.Configuration
	NodesConfig() []NodeConfig
	String() string
}

type configuration struct {
	Nodes                   []NodeConfig         `gluamapper:"nodes"`
	Keys                    Keys                 `gluamapper:"keys"`
	Logging                 logger.Configuration `gluamapper:"logging"`
	HeartbeatIntervalSecond int                  `gluamapper:"heartbeat_interval_second"`
	InfluxDB                InfluxDBConfig       `gluamapper:"influxdb"`
}

//NodeConfig - node config
type NodeConfig struct {
	AddressIPv4   string `gluamapper:"address_ipv4"`
	BroadcastPort string `gluamapper:"broadcast_port"`
	CommandPort   string `gluamapper:"command_port"`
	Chain         string `gluamapper:"chain"`
	Name          string `gluamapper:"name"`
	PublicKey     string `gluamapper:"public_key"`
}

//InfluxDBConfig - influxdb config
type InfluxDBConfig struct {
	Database string `gluamapper:"database"`
	IPv4     string `gluamapper:"ipv4"`
	Port     string `gluamapper:"port"`
	User     string `gluamapper:"user"`
	Password string `gluamapper:"password"`
}

//Keys - public and private keys
type Keys struct {
	Public  string `gluamapper:"public"`
	Private string `gluamapper:"private"`
}

const (
	defaultHeartbeatIntervalSecond = 60
)

var (
	defaultLogging = logger.Configuration{
		Count:     100,
		Console:   false,
		Directory: "log",
		File:      "monitor.log",
		Levels: map[string]string{
			logger.DefaultTag: "error",
		},
		Size: 1048576,
	}
)

//Parse - parse configuration
func Parse(configFile string) (Configuration, error) {
	filePath, err := filepath.Abs(filepath.Clean(configFile))
	if nil != err {
		fmt.Printf("generate file path with error: %s", err)
		return nil, err
	}

	config := &configuration{
		Logging:                 defaultLogging,
		HeartbeatIntervalSecond: defaultHeartbeatIntervalSecond,
	}

	if err := parseLuaConfigurationFile(filePath, config); nil != err {
		fmt.Printf("parse lua config with error: %s", err)
		return nil, err
	}

	return config, nil
}

//Data - return configuration
func (c *configuration) Data() *configuration {
	return c
}

//LogConfig - return log config
func (c *configuration) LogConfig() logger.Configuration {
	return c.Logging
}

//NodesConfig - return nodes config
func (c *configuration) NodesConfig() []NodeConfig {
	return c.Nodes
}

//String - nodes info
func (c *configuration) String() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("Keys:\n\tpublic: \t%s\n\tprivate: \t%s\n", c.Keys.Public, c.Keys.Private))
	str.WriteString("nodes:\n")
	for i, node := range c.Nodes {
		str.WriteString(fmt.Sprintf(
			"\tnode[%d]:\n\t\taddress: \t%s\n\t\tbroadcast port: %s\n\t\tcommand port: \t%s\n\t\tpublic key: \t%s\n\t\tchain: %s\n\t\tname: \t%s\n",
			i,
			node.AddressIPv4,
			node.BroadcastPort,
			node.CommandPort,
			node.PublicKey,
			node.Chain,
			node.Name,
		))
	}
	str.WriteString(fmt.Sprintf("heartbeat interval: %d seconds\n", c.HeartbeatIntervalSecond))
	str.WriteString(fmt.Sprintf("logging: %+v\n", c.Logging))
	str.WriteString("influx database:")
	str.WriteString(fmt.Sprintf("\tip:\t%s\n\tport:\t%s\n\tuser:\t%s\n\tpassword:\t%s\n",
		c.InfluxDB.IPv4,
		c.InfluxDB.Port,
		c.InfluxDB.User,
		c.InfluxDB.Password))
	return str.String()
}

//Key - return key
func (c *configuration) Key() Keys {
	return c.Keys
}

//HeartbeatIntervalInSecond - heartbeat interval in second
func (c *configuration) HeartbeatIntervalInSecond() int {
	return c.HeartbeatIntervalSecond
}

//Influx - return influx config
func (c *configuration) Influx() InfluxDBConfig {
	return c.InfluxDB
}
