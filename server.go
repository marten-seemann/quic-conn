package quicconn

import (
	"crypto/tls"
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

type server struct {
	conn      net.PacketConn
	tlsConfig *tls.Config

	sessionChan chan quic.Session

	quicServer    quic.Listener
	session       quic.Session
	receiveStream quic.Stream
	sendStream    quic.Stream
}

var _ net.Listener = &server{}
var _ net.Conn = &server{}

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
	s.session = <-s.sessionChan

	s.sendStream, err = s.session.OpenStream()
	if err != nil {
		return nil, err
	}

	return s, nil
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

func (s *server) Read(b []byte) (int, error) {
	if s.receiveStream == nil {
		var err error
		s.receiveStream, err = s.session.AcceptStream()
		//TODO: check stream id
		if err != nil {
			return 0, err
		}
	}

	return s.receiveStream.Read(b)
}

func (s *server) Write(b []byte) (int, error) {
	return s.sendStream.Write(b)
}

// LocalAddr returns the local network address.
// needed to fulfill the net.Conn interface
func (s *server) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// Addr returns the listener's network address.
// needed to fulfill the net.Listener interface
func (s *server) Addr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *server) RemoteAddr() net.Addr {
	return nil
}

func (s *server) SetDeadline(t time.Time) error {
	return nil
}

func (s *server) SetReadDeadline(t time.Time) error {
	return nil
}

func (s *server) SetWriteDeadline(t time.Time) error {
	return nil
}
