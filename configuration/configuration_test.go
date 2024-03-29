package configuration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bitmark-inc/logger"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/configuration"
	"github.com/stretchr/testify/assert"
)

const (
	testFile = "test.conf"
)

func setupConfigurationTestFile() {
	content := []byte(`
local M = {}
M.nodes = {
  {
    ip = "127.0.0.1",
    broadcast_port = "1234",
    command_port = "4321",
    public_key = "abcdef",
    chain = "bitmark",
    name = "name1",
  },
  {
    ip = "127.0.0.1",
    broadcast_port = "5678",
    command_port = "8765",
    public_key = "wxyz",
    chain = "testing",
    name = "name2",
  },
}

M.keys = {
  public = "1111",
  private = "2222",
}

M.logging = {
  size = 8888,
  directory = "log",
  count = 10,
  console = false,
  levels = {
      DEFAULT = "error",
  },
}

M.heartbeat_interval_second = 60

M.influxdb = {
  ip = "1.2.3.4",
  port = "5678",
  user = "user",
  password = "password",
  database = "database",
}

M.slack = {
  token = "token",
  channel_id = "channelID",
}

return M
`)

	err := ioutil.WriteFile(testFile, content, 0644)
	if nil != err {
		fmt.Printf("err: %s\n", err)
	}
}

func teardownTestFile() {
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		return
	}

	_ = os.Remove(testFile)
}

func TestParse(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	config, _ := configuration.Parse(testFile)
	actual := config.Data()

	node1 := configuration.NodeConfig{
		IP:            "127.0.0.1",
		BroadcastPort: "1234",
		CommandPort:   "4321",
		PublicKey:     "abcdef",
		Chain:         "bitmark",
		Name:          "name1",
	}

	node2 := configuration.NodeConfig{
		IP:            "127.0.0.1",
		BroadcastPort: "5678",
		CommandPort:   "8765",
		PublicKey:     "wxyz",
		Chain:         "testing",
		Name:          "name2",
	}

	keys := configuration.Keys{
		Public:  "1111",
		Private: "2222",
	}

	influxdb := configuration.InfluxDBConfig{
		IP:       "1.2.3.4",
		Port:     "5678",
		User:     "user",
		Password: "password",
		Database: "database",
	}

	slack := configuration.SlackConfig{
		Token:     "token",
		ChannelID: "channelID",
	}

	assert.Equal(t, keys, actual.Keys, "wrong key")
	assert.Equal(t, 2, len(actual.Nodes), "wrong nodes")
	assert.Equal(t, node1, actual.Nodes[0], "different node info")
	assert.Equal(t, node2, actual.Nodes[1], "different node info")
	assert.Equal(t, influxdb, actual.InfluxDB, "different influxdb")
	assert.Equal(t, slack, actual.Slack, "wrong slack")
}

func TestString(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	config, _ := configuration.Parse(testFile)
	actual := config.String()

	assert.Contains(t, actual, "1111", "wrong public key")
	assert.Contains(t, actual, "2222", "wrong private key")
	assert.Contains(t, actual, "127.0.0.1", "wrong node 1 address")
	assert.Contains(t, actual, "1234", "wrong node 1 broadcast port")
	assert.Contains(t, actual, "4321", "wrong node 1 command port")
	assert.Contains(t, actual, "abcdef", "wrong node 1 public key")
	assert.Contains(t, actual, "bitmark", "wrong node 1 chain")
	assert.Contains(t, actual, "127.0.0.1", "wrong node 2 address")
	assert.Contains(t, actual, "5678", "wrong node 2 broadcast port")
	assert.Contains(t, actual, "8765", "wrong node 2 command port")
	assert.Contains(t, actual, "wxyz", "wrong node 2 public key")
	assert.Contains(t, actual, "testing", "wrong node 2 chain")
	assert.Contains(t, actual, "60", "wrong heartbeat interval")
	assert.Contains(t, actual, "name1", "wrong name")
	assert.Contains(t, actual, "user", "wrong influx user")
	assert.Contains(t, actual, "password", "wrong influx password")
	assert.Contains(t, actual, "token", "wrong slack token")
	assert.Contains(t, actual, "channelID", "wrong slack channel ID")
}

func TestLogging(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	config, _ := configuration.Parse(testFile)
	logConfig := config.LogConfig()

	expected := logger.Configuration{
		Size:      8888,
		Directory: "log",
		Count:     10,
		Console:   false,
		File:      "monitor.log",
		Levels: map[string]string{
			"DEFAULT": "error",
		},
	}

	assert.Equal(t, expected, logConfig, "wrong log config")
}

func TestNodesConfig(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	node1 := configuration.NodeConfig{
		IP:            "127.0.0.1",
		BroadcastPort: "1234",
		CommandPort:   "4321",
		PublicKey:     "abcdef",
		Chain:         "bitmark",
		Name:          "name1",
	}

	node2 := configuration.NodeConfig{
		IP:            "127.0.0.1",
		BroadcastPort: "5678",
		CommandPort:   "8765",
		PublicKey:     "wxyz",
		Chain:         "testing",
		Name:          "name2",
	}

	config, _ := configuration.Parse(testFile)
	nodes := config.NodesConfig()

	assert.Equal(t, 2, len(nodes), "wrong nodes count")
	assert.Equal(t, node1, nodes[0], "wrong node1")
	assert.Equal(t, node2, nodes[1], "wrong node2")
}

func TestKey(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	expected := configuration.Keys{
		Public:  "1111",
		Private: "2222",
	}

	config, _ := configuration.Parse(testFile)
	actual := config.Key()

	assert.Equal(t, expected, actual, "wrong key")
}

func TestHeartbeatIntervalInSecond(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	config, _ := configuration.Parse(testFile)
	heartbeatInterval := config.HeartbeatIntervalInSecond()

	assert.Equal(t, 60, heartbeatInterval, "wrong heartbeat interval")
}

func TestInfluxDB(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	config, _ := configuration.Parse(testFile)
	influxdb := config.Influx()
	expected := configuration.InfluxDBConfig{
		IP:       "1.2.3.4",
		Port:     "5678",
		User:     "user",
		Password: "password",
		Database: "database",
	}

	assert.Equal(t, expected, influxdb, "wrong influxdb")
}

func TestSlack(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	config, _ := configuration.Parse(testFile)
	slack := config.SlackConfig()
	expected := configuration.SlackConfig{
		Token:     "token",
		ChannelID: "channelID",
	}

	assert.Equal(t, expected, slack, "wrong influxdb")
}
