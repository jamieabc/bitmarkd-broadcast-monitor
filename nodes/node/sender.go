package node

import "github.com/jamieabc/bitmarkd-broadcast-monitor/communication"

func senderLoop(n Node, shutdown <-chan struct{}) {
	log := n.Log()
loop:
	for {
		select {
		case <-shutdown:
			log.Infof("receive shutdown signal")
			break loop
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
