package communication

import "github.com/jamieabc/bitmarkd-broadcast-monitor/network"

//Communication - communication between bitmarkd
type Communication interface {
	Get(...interface{}) (interface{}, error)
}

//ComType - communication type
type ComType int

const (
	//ComInfo - communication for info
	ComInfo ComType = iota

	//ComDigest - communication for digest
	ComDigest
)

//New - new communication
func New(comType ComType, client network.Client) Communication {
	switch comType {
	case ComInfo:
		return newInfo(client)
	case ComDigest:
		return newDigest(client)
	}
	return nil
}
