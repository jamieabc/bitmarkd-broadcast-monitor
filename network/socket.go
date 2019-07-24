package network

import (
	"time"

	zmq "github.com/pebbe/zmq4"
)

// point at which to disconnect large message senders
// current estimate of a block maximum is 2 MB
const (
	maximumPacketSize = 5000000 // 5 MB
)

// ***** FIX THIS: enabling this causes complete failure
// ***** FIX THIS: socket disconnects, perhaps after IVL value
// const (
// 	heartbeatInterval = 15 * time.Second
// 	heartbeatTimeout  = 60 * time.Second
// 	heartbeatTTL      = 60 * time.Second
// )

// return a pair of connected PAIR sockets
// for shutdown signalling
func NewSignalPair(signal string) (reciever *zmq.Socket, sender *zmq.Socket, err error) {

	// PAIR server, half of signalling channel
	reciever, err = zmq.NewSocket(zmq.PAIR)
	if nil != err {
		return nil, nil, err
	}
	_ = reciever.SetLinger(0)
	err = reciever.Bind(signal)
	if nil != err {
		_ = reciever.Close()
		return nil, nil, err
	}

	// PAIR Client, half of signalling channel
	sender, err = zmq.NewSocket(zmq.PAIR)
	if nil != err && nil != sender {
		_ = sender.Close()
		return nil, nil, err
	}
	_ = sender.SetLinger(0)
	err = sender.Connect(signal)
	if nil != err {
		_ = sender.Close()
		return nil, nil, err
	}

	return reciever, sender, nil
}

// create a socket suitable for a server side connection
func NewServerSocket(socketType zmq.Type, zapDomain string, privateKey []byte, publicKey []byte, v6 bool) (*zmq.Socket, error) {

	socket, err := zmq.NewSocket(socketType)
	if nil != err {
		return nil, err
	}

	// allow any client to connect
	//zmq.AuthAllow(zapDomain, "127.0.0.1/8")
	//zmq.AuthAllow(zapDomain, "::1")
	zmq.AuthCurveAdd(zapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(string(privateKey))

	err = socket.SetZapDomain(zapDomain)
	if nil != err {
		goto failure
	}

	socket.SetIdentity(string(publicKey)) // just use public key for identity

	err = socket.SetIpv6(v6) // conditionally set IPv6 state
	if nil != err {
		goto failure
	}

	// only queue message to connected peers
	socket.SetImmediate(true)
	socket.SetLinger(100 * time.Millisecond)

	err = socket.SetSndtimeo(120 * time.Second)
	if nil != err {
		goto failure
	}
	err = socket.SetRcvtimeo(120 * time.Second)
	if nil != err {
		goto failure
	}

	err = socket.SetTcpKeepalive(1)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveCnt(5)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveIdle(60)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveIntvl(60)
	if nil != err {
		goto failure
	}

	// ***** FIX THIS: enabling this causes complete failure
	// ***** FIX THIS: socket disconnects, perhaps after IVL value
	// heartbeat
	// err = socket.SetHeartbeatIvl(heartbeatInterval)
	// if nil != err {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTimeout(heartbeatTimeout)
	// if nil != err {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTtl(heartbeatTTL)
	// if nil != err {
	// 	goto failure
	// }

	err = socket.SetMaxmsgsize(maximumPacketSize)
	if nil != err {
		goto failure
	}

	return socket, nil

failure:
	return nil, err
}
