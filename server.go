package quicconn

import (
	"crypto/tls"
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/utils"
)

type server struct {
	conn       *net.UDPConn
	quicServer *quic.Server
	dataStream utils.Stream
}

var _ Listener = &server{}
var _ net.Conn = &server{}

// Accept waits for and returns the next connection to the listener.
func (s *server) Accept(sni string, tlsConfig *tls.Config) (net.Conn, error) {
	c := make(chan utils.Stream, 1)

	cb := func(_ *quic.Session, stream utils.Stream) {
		if stream.StreamID() != 1 {
			c <- stream
		}
	}

	quicServer, err := quic.NewServer(sni, tlsConfig, cb)
	if err != nil {
		return nil, err
	}
	go quicServer.Serve(s.conn)
	s.quicServer = quicServer
	// wait until a client establishes a connection
	s.dataStream = <-c
	return s, nil
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

func (s *server) Read(b []byte) (int, error) {
	return s.dataStream.Read(b)
}

func (s *server) Write(b []byte) (int, error) {
	return s.dataStream.Write(b)
}

func (s *server) LocalAddr() net.Addr {
	return nil
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
