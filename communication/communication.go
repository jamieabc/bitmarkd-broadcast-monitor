package communication

import "github.com/jamieabc/bitmarkd-broadcast-monitor/network"

//Communication - communication between bitmarkd
type Communication interface {
	Get(...interface{}) (interface{}, error)
}

//ComType - communication type
type ComType int

const (
	// ComInfo - communication for info
	ComInfo ComType = iota

	// ComHeight - communication for height
	ComHeight

	// ComBlockHeader - communication for block
	ComBlockHeader
)

//New - new communication
func New(comType ComType, client network.Client) Communication {
	switch comType {
	case ComInfo:
		return newInfo(client)
	case ComHeight:
		return newHeight(client)
	case ComBlockHeader:
		return newBlock(client)
	}
	return nil
}
