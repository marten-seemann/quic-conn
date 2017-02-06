package quicconn

import (
	"crypto/tls"
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/utils"
)

type client struct {
	quicClient *quic.Client
	dataStream utils.Stream
}

var _ net.Conn = &client{}

func newClient(addr string, tlsConfig *tls.Config) (*client, error) {
	c := make(chan struct{}, 1)
	versionNegotiateCallback := func() error { return nil }
	cryptoChangeCallback := func(isForwardSecure bool) {
		if isForwardSecure {
			c <- struct{}{}
		}
	}
	quicClient, err := quic.NewClient(addr, tlsConfig, cryptoChangeCallback, versionNegotiateCallback)
	if err != nil {
		return nil, err
	}
	go quicClient.Listen()

	// wait for the crypto handshake to complete
	<-c
	// open the data stream
	dataStream, err := quicClient.OpenStream(3)
	if err != nil {
		return nil, err
	}

	return &client{
		quicClient: quicClient,
		dataStream: dataStream,
	}, nil
}

func (c *client) Close() error {
	return c.quicClient.Close(nil)
}

func (c *client) Read(b []byte) (int, error) {
	return c.dataStream.Read(b)
}

func (c *client) Write(b []byte) (int, error) {
	return c.dataStream.Write(b)
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
