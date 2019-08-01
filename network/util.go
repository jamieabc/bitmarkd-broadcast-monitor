package network

import (
	"net"
	"strconv"
	"strings"

	"github.com/jamieabc/bitmarkd-broadcast-monitor/fault"
)

const (
	minPort = 1
	maxPort = 65535
)

// Connection - remote ip and port
type Connection struct {
	ip   net.IP
	port uint16
}

// NewConnection - new connection
func NewConnection(hostPort string) (*Connection, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if nil != err {
		return nil, fault.InvalidIPAddress
	}

	IP := net.ParseIP(strings.Trim(host, " "))
	if nil == IP {
		ips, err := net.LookupIP(host)
		if nil != err {
			return nil, err
		}
		if len(ips) < 1 {
			return nil, fault.InvalidIPAddress
		}
		IP = ips[0]
	}

	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return nil, err
	}
	if numericPort < minPort || numericPort > maxPort {
		return nil, fault.InvalidPortNumber
	}
	c := &Connection{
		ip:   IP,
		port: uint16(numericPort),
	}
	return c, nil
}

// canonicalIPandPort - make IP:Port into canonical string, returns prefixed string and IPv6 flag
// prefix is optional and can be empty ("")
// IPv4:  127.0.0.1:1234
// IPv6:  [::1]:1234
func (c *Connection) canonicalIPandPort(prefix string) (string, bool) {
	port := int(c.port)
	if nil != c.ip.To4() {
		return prefix + c.ip.String() + ":" + strconv.Itoa(port), false
	}
	return prefix + "[" + c.ip.String() + "]:" + strconv.Itoa(port), true
}
