package network

import (
	"sync"

	zmq "github.com/pebbe/zmq4"
)

// to ensure only one auth start
var oneTimeAuthStart sync.Once

// initilaise the ZMQ security subsystem
func StartAuthentication() error {

	err := error(nil)
	oneTimeAuthStart.Do(func() {

		// initialise encryption
		zmq.AuthSetVerbose(false)
		//zmq.AuthSetVerbose(true)
		err = zmq.AuthStart()
	})

	return err
}
