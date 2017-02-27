package quicconn

import (
	"crypto/tls"
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

type client struct {
	session       quic.Session
	receiveStream quic.Stream
	sendStream    quic.Stream
}

var _ net.Conn = &client{}

func newClient(addr string, tlsConfig *tls.Config) (*client, error) {
	config := &quic.Config{
		TLSConfig: tlsConfig,
	}

	quicSession, err := quic.DialAddr(addr, config)
	if err != nil {
		return nil, err
	}

	sendStream, err := quicSession.OpenStream()
	if err != nil {
		return nil, err
	}

	return &client{
		session:    quicSession,
		sendStream: sendStream,
	}, nil
}

func (c *client) Close() error {
	return c.session.Close(nil)
}

func (c *client) Read(b []byte) (int, error) {
	if c.receiveStream == nil {
		var err error
		c.receiveStream, err = c.session.AcceptStream()
		if err != nil {
			return 0, err
		}
	}
	return c.receiveStream.Read(b)
}

func (c *client) Write(b []byte) (int, error) {
	return c.sendStream.Write(b)
}

func (c *client) LocalAddr() net.Addr {
	return nil
}

func (c *client) RemoteAddr() net.Addr {
	return nil
}

func (c *client) SetDeadline(t time.Time) error {
	return nil
}

func (c *client) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *client) SetWriteDeadline(t time.Time) error {
	return nil
}
