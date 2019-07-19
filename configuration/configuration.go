package configuration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitmark-inc/logger"
)

type Configuration interface {
	Data() *ConfigurationImpl
	HeartbeatIntervalInSecond() int
	LogConfig() logger.Configuration
	NodesConfig() []NodeConfig
	Key() Keys
	String() string
}

type ConfigurationImpl struct {
	Nodes                   []NodeConfig         `gluamapper:"nodes"`
	Keys                    Keys                 `gluamapper:"keys"`
	Logging                 logger.Configuration `gluamapper:"logging"`
	HeartbeatIntervalSecond int                  `gluamapper:"heartbeat_interval_second"`
}

type NodeConfig struct {
	AddressIPv4   string `gluamapper:"address_ipv4"`
	BroadcastPort string `gluamapper:"broadcast_port"`
	CommandPort   string `gluamapper:"command_port"`
	Chain         string `gluamapper:"chain"`
	PublicKey     string `gluamapper:"public_key"`
}

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

// Parse - parse configuration
func Parse(configFile string) (Configuration, error) {
	filePath, err := filepath.Abs(filepath.Clean(configFile))
	if nil != err {
		return nil, err
	}

	config := &ConfigurationImpl{
		Logging:                 defaultLogging,
		HeartbeatIntervalSecond: defaultHeartbeatIntervalSecond,
	}

	if err := parseLuaConfigurationFile(filePath, config); nil != err {
		return nil, err
	}

	return config, nil
}

// Data - return configuration
func (c *ConfigurationImpl) Data() *ConfigurationImpl {
	return c
}

// LogConfig - return log config
func (c *ConfigurationImpl) LogConfig() logger.Configuration {
	return c.Logging
}

// NodesConfig - return nodes config
func (c *ConfigurationImpl) NodesConfig() []NodeConfig {
	return c.Nodes
}

// String - nodes info
func (c *ConfigurationImpl) String() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("Keys:\n\tpublic: \t%s\n\tprivate: \t%s\n", c.Keys.Public, c.Keys.Private))
	str.WriteString("nodes:\n")
	for i, node := range c.Nodes {
		str.WriteString(fmt.Sprintf(
			"\tnode[%d]:\n\t\taddress: \t%s\n\t\tbroadcast port: %s\n\t\tcommand port: \t%s\n\t\tpublic key: \t%s\n\t\tchain: %s\n",
			i,
			node.AddressIPv4,
			node.BroadcastPort,
			node.CommandPort,
			node.PublicKey,
			node.Chain,
		))
	}
	str.WriteString(fmt.Sprintf("heartbeat interval: %d seconds\n", c.HeartbeatIntervalSecond))
	str.WriteString(fmt.Sprintf("logging: %+v\n", c.Logging))
	return str.String()
}

// Key - return key
func (c *ConfigurationImpl) Key() Keys {
	return c.Keys
}

// HeartbeatIntervalInSecond - heartbeat interval in second
func (c *ConfigurationImpl) HeartbeatIntervalInSecond() int {
	return c.HeartbeatIntervalSecond
}
