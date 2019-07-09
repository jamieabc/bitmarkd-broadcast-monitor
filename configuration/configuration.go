// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"fmt"
	"path/filepath"
)

type Configuration struct {
	Nodes []Node `gluamapper:"nodes"`
	Keys  Keys   `gluamapper:"keys"`
}

type Node struct {
	AddressIPv4 string `gluamapper:"address_ipv4" json:"address_ipv4"`
	PublicKey   string `gluamapper:"public_key" json:"public_key"`
}

type Keys struct {
	Public  string `gluamapper:"public"`
	Private string `gluamapper:"private"`
}

// Parse - parse configuration
func Parse(configFile string) (*Configuration, error) {
	filePath, err := filepath.Abs(filepath.Clean(configFile))
	if nil != err {
		return nil, err
	}

	directory, _ := filepath.Split(filePath)

	fmt.Printf("dir: %s, file name: %s\n", directory, filePath)

	config := &Configuration{}

	if err := parseLuaConfigurationFile(filePath, config); nil != err {
		return nil, err
	}

	return config, nil
}
