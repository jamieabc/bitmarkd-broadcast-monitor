package fault

import "errors"

var (
	// InvalidIPAddress - invalid ip address
	InvalidIPAddress = errors.New("invalid ip Address")

	// InvalidPortNumber - invalid port number
	InvalidPortNumber = errors.New("invalid port number")

	// InvalidArguments - invalid arguments
	InvalidArguments = errors.New("invalid arguments")

	// InvalidEmptyConfigFile - invalid empty config file
	InvalidEmptyConfigFile = errors.New("empty config file")

	// InvalidConnection - invalid ip or port
	InvalidConnection = errors.New("ip or port invalid")

	// InvalidPublicKeyFile - invalid public key file
	InvalidPublicKeyFile = errors.New("invalid public key file")

	// InvalidPrivateKeyFile - invalid private key file
	InvalidPrivateKeyFile = errors.New("invalid private key file")

	// InsufficientSlackSendParameter - insufficient slack send parameter
	InsufficientSlackSendParameter = errors.New("insufficient slack send parameter")
)
