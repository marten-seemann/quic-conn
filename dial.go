package quicconn

import (
	"crypto/tls"
	"net"
)

// Listen announces on the local network address laddr.
func Listen(network, laddr string) (Listener, error) {
	udpAddr, err := net.ResolveUDPAddr(network, laddr)
	if err != nil {
		return nil, &net.OpError{Op: "listen", Net: network, Source: nil, Addr: nil, Err: err}
	}
	conn, err := net.ListenUDP(network, udpAddr)
	return &server{conn: conn}, err
}

// Dial creates a new QUIC connection
// it returns once the connection is established and secured with forward-secure keys
func Dial(addr string, tlsConfig *tls.Config) (net.Conn, error) {
	c, err := newClient(addr, tlsConfig)
	if err != nil {
		return nil, err
	}
	return c, nil
}
