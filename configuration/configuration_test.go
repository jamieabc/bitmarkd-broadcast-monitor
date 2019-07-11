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
    address_ipv4 = "127.0.0.1:1234",
    public_key = "abcdef",
    chain = "bitmark",
  },
  {
    address_ipv4 = "127.0.0.1:5678",
    public_key = "wxyz",
    chain = "testing",
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

	os.Remove(testFile)
}

func TestParse(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	assert := assert.New(t)

	config, err := configuration.Parse(testFile)
	actual := config.Data()

	node1 := configuration.NodeConfig{
		AddressIPv4: "127.0.0.1:1234",
		PublicKey:   "abcdef",
		Chain:       "bitmark",
	}

	node2 := configuration.NodeConfig{
		AddressIPv4: "127.0.0.1:5678",
		PublicKey:   "wxyz",
		Chain:       "testing",
	}

	keys := configuration.Keys{
		Public:  "1111",
		Private: "2222",
	}

	assert.Equal(nil, err, "parse fail")
	assert.Equal(keys, actual.Keys, "wrong key")
	assert.Equal(2, len(actual.Nodes), "wrong nodes")
	assert.Equal(node1, actual.Nodes[0], "different node info")
	assert.Equal(node2, actual.Nodes[1], "different node info")
}

func TestString(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	config, _ := configuration.Parse(testFile)
	actual := config.String()

	assert.Contains(t, actual, "1111", "wrong public key")
	assert.Contains(t, actual, "2222", "wrong private key")
	assert.Contains(t, actual, "127.0.0.1:1234", "wrong node 1 address")
	assert.Contains(t, actual, "abcdef", "wrong node 1 public key")
	assert.Contains(t, actual, "bitmark", "wrong node 1 chain")
	assert.Contains(t, actual, "127.0.0.1:5678", "wrong node 2 address")
	assert.Contains(t, actual, "wxyz", "wrong node 2 public key")
	assert.Contains(t, actual, "testing", "wrong node 2 chain")
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
		Levels: map[string]string{
			"DEFAULT": "error",
		},
	}

	assert.Equal(t, expected, logConfig, ("wrong log config"))
}

func TestNodesConfig(t *testing.T) {
	setupConfigurationTestFile()
	defer teardownTestFile()

	node1 := configuration.NodeConfig{
		AddressIPv4: "127.0.0.1:1234",
		PublicKey:   "abcdef",
		Chain:       "bitmark",
	}

	node2 := configuration.NodeConfig{
		AddressIPv4: "127.0.0.1:5678",
		PublicKey:   "wxyz",
		Chain:       "testing",
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
