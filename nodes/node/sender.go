package node

import (
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/jamieabc/bitmarkd-broadcast-monitor/communication"
)

func senderLoop(n Node, shutdown <-chan struct{}, notifyChan chan struct{}) {
	log := n.Log()
loop:
	for {
		select {
		case <-shutdown:
			log.Infof("receive shutdown signal")
			break loop
		case <-notifyChan:
			digest, height, err := remoteDigestOfHeight(n)
			if nil != err {
				log.Errorf("get remote digest of height with error: %s", err)
				return
			}
			log.Infof("remote height %d with digest %s", height, digest)
		case <-n.CheckTimer().C:
			log.Debug("time to check remote block")
			info, err := remoteInfo(n)
			if nil != err {
				log.Errorf("get remote info error: %s", err)
				log.Infof("remote info: %v\n", info)
			}
			log.Infof("remote info: %s", info)
			n.CheckTimer().Reset(checkIntervalSecond)
		}
	}
	log.Infof("finish")
}

func remoteInfo(n Node) (*communication.InfoResponse, error) {
	info, err := n.Client().Info()
	if nil != err {
		return nil, err
	}
	return info, nil
}

func remoteDigestOfHeight(n Node) (blockdigest.Digest, uint64, error) {
	info, err := n.Client().Info()
	if nil != err {
		return blockdigest.Digest{}, uint64(0), err
	}
	height := info.Height
	digest, err := n.Client().DigestOfHeight(height)
	if nil != err {
		return blockdigest.Digest{}, uint64(0), err
	}
	return digest.Digest, height, nil
}
