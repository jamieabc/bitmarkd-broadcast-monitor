package node

import (
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/communication"
)

func senderLoop(n Node) {
	log := n.Log()
loop:
	for {
		select {
		case <-shutdownChan:
			log.Infof("terminate sender loop")
			break loop
		case <-notifyChan:
			height, err := remoteHeight(n)
			if nil != err {
				log.Errorf("get remote height with error: %s", err)
				continue
			}
			digest, err := remoteDigestOfHeight(n, height)
			if nil != err {
				log.Errorf("get remote digest of height with error: %s", err)
				continue
			}
			log.Infof("remote height %d with digest %s", height, digest)
		case <-n.CheckTimer().C:
			info, err := remoteInfo(n)
			if nil != err {
				log.Errorf("get remote info error: %s", err)
				continue
			}
			log.Infof("remote info: %s", info)
			digest, err := remoteDigestOfHeight(n, info.Height)
			if nil != err {
				log.Errorf("remote height %d digest with error: %s", info.Height, err)
				continue
			}
			log.Infof("remote height %d digest %s", info.Height, digest)
			n.CheckTimer().Reset(checkIntervalSecond)
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

func remoteDigestOfHeight(n Node, height uint64) (blockdigest.Digest, error) {
	digest, err := n.Remote().DigestOfHeight(height)
	if nil != err {
		return blockdigest.Digest{}, err
	}
	return digest.Digest, nil
}
