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

	quicServer quic.Listener
}

var _ net.Listener = &server{}

// Accept waits for and returns the next connection to the listener.
func (s *server) Accept() (net.Conn, error) {
	s.sessionChan = make(chan quic.Session)

	config := &quic.Config{
		TLSConfig: s.tlsConfig,
		ConnState: s.connStateCallback,
	}

	quicServer, err := quic.Listen(s.conn, config)
	if err != nil {
		return nil, err
	}
	go quicServer.Serve()
	s.quicServer = quicServer

	// wait until a client establishes a connection
	sess := <-s.sessionChan
	qconn, err := newConn(sess)
	if err != nil {
		return nil, err
	}
	return qconn, nil
}

func (s *server) connStateCallback(sess quic.Session, state quic.ConnState) {
	if state == quic.ConnStateForwardSecure {
		s.sessionChan <- sess
	}
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
