package fault

import "errors"

var (
	InvalidIPAddress  = errors.New("invalid ip Address")
	InvalidPortNumber = errors.New("invalid port number")
)
