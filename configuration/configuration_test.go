// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

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
  },
  {
    address_ipv4 = "127.0.0.1:5678",
    public_key = "wxyz",
  },
}
M.keys = {
  public = "1111",
  private = "2222",
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

	actual, err := configuration.Parse(testFile)

	node1 := configuration.Node{
		AddressIPv4: "127.0.0.1:1234",
		PublicKey:   "abcdef",
	}

	node2 := configuration.Node{
		AddressIPv4: "127.0.0.1:5678",
		PublicKey:   "wxyz",
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
