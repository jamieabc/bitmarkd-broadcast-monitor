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
			digest, height, err := remoteDigestOfHeight(n)
			if nil != err {
				log.Errorf("get remote digest of height with error: %s", err)
				continue
			}
			log.Infof("remote height %d with digest %s", height, digest)
		case <-n.CheckTimer().C:
			info, err := remoteInfo(n)
			if nil != err {
				log.Errorf("get remote info error: %s", err)
				log.Infof("remote info: %v\n", info)
				continue
			}
			log.Infof("remote info: %s", info)
			n.CheckTimer().Reset(checkIntervalSecond)
		}
	}
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
