package network

import (
	"encoding/hex"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
)

const (
	taggedPublic  = "PUBLIC:"
	taggedPrivate = "PRIVATE:"
	publicLength  = 32
	privateLength = 32
)

// read a public key from a string returning it as a 32 byte string
func ReadPublicKey(key string) ([]byte, error) {
	data, private, err := ParseKey(key)
	if err != nil {
		return []byte{}, err
	}
	if private {
		return []byte{}, fault.ErrInvalidPublicKeyFile
	}
	return data, err
}

// read a private key from a string returning it as a 32 byte string
func ReadPrivateKey(key string) ([]byte, error) {
	data, private, err := ParseKey(key)
	if err != nil {
		return []byte{}, err
	}
	if !private {
		return []byte{}, fault.ErrInvalidPrivateKeyFile
	}
	return data, err
}

func ParseKey(data string) ([]byte, bool, error) {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, taggedPrivate) {
		h, err := hex.DecodeString(s[len(taggedPrivate):])
		if err != nil {
			return []byte{}, false, err
		}
		if len(h) != privateLength {
			return []byte{}, false, fault.ErrInvalidPrivateKeyFile
		}
		return h, true, nil
	} else if strings.HasPrefix(s, taggedPublic) {
		h, err := hex.DecodeString(s[len(taggedPublic):])
		if err != nil {
			return []byte{}, false, err
		}
		if len(h) != publicLength {
			return []byte{}, false, fault.ErrInvalidPublicKeyFile
		}
		return h, false, nil
	}

	return []byte{}, false, fault.ErrInvalidPublicKeyFile
}
