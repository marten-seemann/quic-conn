package quicconn

import (
	"crypto/tls"
	"net"

	quic "github.com/lucas-clemente/quic-go"
)

type server struct {
	conn      net.PacketConn
	tlsConfig *tls.Config

	sessionChan chan quic.Session
	errorChan   chan error

	quicServer quic.Listener
}

var _ net.Listener = &server{}

// Accept waits for and returns the next connection to the listener.
func (s *server) Accept() (net.Conn, error) {
	config := &quic.Config{
		TLSConfig: s.tlsConfig,
	}

	quicServer, err := quic.Listen(s.conn, config)
	if err != nil {
		return nil, err
	}
	s.quicServer = quicServer

	// wait until a client establishes a connection
	sess, err := quicServer.Accept()
	if err != nil {
		return nil, err
	}
	qconn, err := newConn(sess)
	if err != nil {
		return nil, err
	}
	return qconn, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (s *server) Close() error {
	return s.quicServer.Close()
}

// Addr returns the listener's network address.
func (s *server) Addr() net.Addr {
	return s.conn.LocalAddr()
}
