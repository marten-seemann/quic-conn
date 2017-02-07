package quicconn

import (
	"crypto/tls"
	"net"
)

// Listen creates a QUIC listener on the given network interface
func Listen(network, laddr string, tlsConfig *tls.Config) (Listener, error) {
	udpAddr, err := net.ResolveUDPAddr(network, laddr)
	if err != nil {
		return nil, &net.OpError{Op: "listen", Net: network, Source: nil, Addr: nil, Err: err}
	}
	conn, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return nil, err
	}
	return &server{
		conn:      conn,
		tlsConfig: tlsConfig,
	}, nil
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
