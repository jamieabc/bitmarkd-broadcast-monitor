package node

import (
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockrecord"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/communication"
)

const (
	checkIntervalSecond = 5 * time.Minute
)

func senderLoop(args []interface{}) {
	if 1 != len(args) {
		fmt.Println("senderLoop wrong argument length")
		return
	}
	n := args[0].(Node)
	log := n.Log()
	timer := time.NewTimer(checkIntervalSecond)

	for {
		select {
		case <-ctx.Done():
			log.Infof("terminate sender loop")
			return

		case <-timer.C:
			info, err := remoteInfo(n)
			if nil != err {
				log.Errorf("get remote info error: %s", err)
				continue
			}
			log.Infof("remote info: %s", info)
			header, digest, err := remoteBlockHeader(n, info.Height)
			log.Infof(
				"remote height %d with digest %s, generated at %s",
				info.Height,
				digest,
				time.Unix(int64(header.Timestamp), 0),
			)
			timer.Reset(checkIntervalSecond)
		}
	}
}

func remoteInfo(n Node) (*communication.InfoResponse, error) {
	info, err := n.Remote().Info()
	if nil != err {
		return nil, err
	}
	return info, nil
}

func remoteHeight(n Node) (uint64, error) {
	height, err := n.Remote().Height()
	if nil != err {
		return uint64(0), err
	}
	return height.Height, nil
}

func remoteBlockHeader(n Node, height uint64) (*blockrecord.Header, blockdigest.Digest, error) {
	resp, err := n.Remote().BlockHeader(height)
	if nil != err {
		return &blockrecord.Header{}, blockdigest.Digest{}, err
	}
	return resp.Header, resp.Digest, nil
}
