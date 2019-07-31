package communication

import (
	"encoding/binary"
	"fmt"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/network"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/fault"
)

type digest struct {
	client network.Client
	prefix string
}

func newDigest(client network.Client) Communication {
	return &digest{
		client: client,
		prefix: "I",
	}
}

// Get - get digest
func (d *digest) Get(args ...interface{}) (interface{}, error) {
	if 1 < len(args) {
		return nil, fault.InvalidArguments
	}

	arg := args[0]

	err := d.client.Send(d.prefix, genPayload(arg.(uint64)))
	if nil != err {
		return nil, err
	}

	data, err := d.client.Receive(0)
	if nil != err {
		return nil, err
	}

	if d.prefix != string(data[0]) || 2 != len(data) {
		return nil, fmt.Errorf("wrong command")
	}

	digest := blockdigest.Digest{}
	err = blockdigest.DigestFromBytes(&digest, data[1])
	if nil != err {
		return nil, err
	}
	fmt.Printf("digest: %s\n", digest.String())

	return &digest, nil
}

func genPayload(height uint64) []byte {
	params := make([]byte, 8)
	binary.BigEndian.PutUint64(params, height)
	return params
}
