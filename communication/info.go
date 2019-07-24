package communication

import (
	"encoding/json"
	"fmt"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
)

type info struct {
	client *network.Client
	prefix string
}

type InfoResponse struct {
	Version string `json:"version"`
	Chain   string `json:"chain"`
	Normal  bool   `json:"normal"`
	Height  uint64 `json:"height"`
}

func newInfo(client *network.Client) Communication {
	return &info{
		client: client,
		prefix: "I",
	}
}

// Get - get info
func (i *info) Get(payload ...interface{}) (interface{}, error) {
	err := i.client.Send(i.prefix)
	if nil != err {
		return nil, err
	}

	data, err := i.client.Receive(0)
	if nil != err {
		return nil, err
	}

	if i.prefix != string(data[0]) {
		return nil, fmt.Errorf("wrong command")
	}

	var info InfoResponse
	err = json.Unmarshal(data[1], &info)
	if nil != err {
		return nil, err
	}
	fmt.Printf("info: %+v\n", info)

	return &info, nil
}
