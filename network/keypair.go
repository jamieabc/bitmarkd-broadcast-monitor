package network

import (
	"encoding/hex"
	"strings"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/fault"
)

const (
	taggedPublic  = "PUBLIC:"
	taggedPrivate = "PRIVATE:"
	publicLength  = 32
	privateLength = 32
)

// read a public key from a string returning it as a 32 byte string
func ReadPublicKey(key string) ([]byte, error) {
	data, private, err := parseKey(key)
	if err != nil {
		return []byte{}, err
	}
	if private {
		return []byte{}, fault.InvalidPublicKeyFile
	}
	return data, err
}

// read a private key from a string returning it as a 32 byte string
func ReadPrivateKey(key string) ([]byte, error) {
	data, private, err := parseKey(key)
	if err != nil {
		return []byte{}, err
	}
	if !private {
		return []byte{}, fault.InvalidPrivateKeyFile
	}
	return data, err
}

// parseKey - parse key
func parseKey(data string) ([]byte, bool, error) {
	s := strings.TrimSpace(data)
	if strings.HasPrefix(s, taggedPrivate) {
		h, err := hex.DecodeString(s[len(taggedPrivate):])
		if err != nil {
			return []byte{}, false, err
		}
		if len(h) != privateLength {
			return []byte{}, false, fault.InvalidPrivateKeyFile
		}
		return h, true, nil
	} else if strings.HasPrefix(s, taggedPublic) {
		h, err := hex.DecodeString(s[len(taggedPublic):])
		if err != nil {
			return []byte{}, false, err
		}
		if len(h) != publicLength {
			return []byte{}, false, fault.InvalidPublicKeyFile
		}
		return h, false, nil
	}

	return []byte{}, false, fault.InvalidPublicKeyFile
}
