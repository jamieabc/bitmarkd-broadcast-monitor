package communication

import (
	"encoding/binary"
	"fmt"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"
)

type height struct {
	client network.Client
	prefix string
}

//HeightResponse - json response of height command
type HeightResponse struct {
	Height uint64
}

//Get - get height
func (h *height) Get(payload ...interface{}) (interface{}, error) {
	err := h.client.Send(h.prefix)
	if nil != err {
		return nil, err
	}

	data, err := h.client.Receive(0)
	if nil != err {
		return nil, err
	}

	if h.prefix != string(data[0]) {
		return nil, fmt.Errorf("wrong command")
	}

	return &HeightResponse{Height: binary.BigEndian.Uint64(data[1])}, nil
}

func newHeight(client network.Client) Communication {
	return &height{
		client: client,
		prefix: "N",
	}
}
