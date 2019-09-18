package communication

import (
	"encoding/binary"
	"fmt"

	"github.com/bitmark-inc/bitmarkd/blockdigest"

	"github.com/bitmark-inc/bitmarkd/blockrecord"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/fault"
)

type block struct {
	client network.Client
	prefix string
}

// BlockHeaderResponse - block response
type BlockHeaderResponse struct {
	Digest blockdigest.Digest
	Header *blockrecord.Header
}

func newBlock(client network.Client) Communication {
	return &block{
		client: client,
		prefix: "B",
	}
}

// Get - get block header
func (b *block) Get(args ...interface{}) (interface{}, error) {
	if 1 < len(args) {
		return nil, fault.InvalidArguments
	}

	height := args[0].(uint64)
	params := make([]byte, 8)
	binary.BigEndian.PutUint64(params, height)

	err := b.client.Send(b.prefix, params)
	if nil != err {
		return nil, err
	}

	data, err := b.client.Receive(0)
	if nil != err {
		return nil, err
	}

	if b.prefix != string(data[0]) || 2 != len(data) {
		return nil, fmt.Errorf("wrong command")
	}

	header, digest, _, err := blockrecord.ExtractHeader(data[1], uint64(0))
	if nil != err {
		return nil, err
	}

	return &BlockHeaderResponse{
		Digest: digest,
		Header: header,
	}, nil
}
