package communication

import "github.com/jamieabc/bitmarkd-broadcast-monitor/network"

type Communication interface {
	Get(...interface{}) (interface{}, error)
}

type ComType int

const (
	ComInfo ComType = iota
	ComDigest
)

// New - new communication
func New(comType ComType, client *network.Client) Communication {
	switch comType {
	case ComInfo:
		return newInfo(client)
	case ComDigest:
		return newDigest(client)
	}
	return nil
}
