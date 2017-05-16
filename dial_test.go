package quicconn

import (
	"crypto/tls"
	"errors"
	"net"

	quic "github.com/lucas-clemente/quic-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dial and Listen", func() {
	AfterEach(func() {
		quicListen = quic.Listen
	})

	It("listens", func() {
		var conn net.PacketConn
		var conf *quic.Config
		tlsConf := &tls.Config{}
		quicListen = func(c net.PacketConn, quicConfig *quic.Config) (quic.Listener, error) {
			conn = c
			conf = quicConfig
			return nil, nil
		}
		_, err := Listen("udp", "localhost:12345", tlsConf)
		Expect(err).ToNot(HaveOccurred())
		Expect(conn.(*net.UDPConn).LocalAddr().String()).To(Equal("127.0.0.1:12345"))
		Expect(conf.TLSConfig).To(Equal(tlsConf))
	})

	It("returns listen errors", func() {
		testErr := errors.New("listen error")
		quicListen = func(_ net.PacketConn, _ *quic.Config) (quic.Listener, error) {
			return nil, testErr
		}
		_, err := Listen("udp", "localhost:12346", &tls.Config{})
		Expect(err).To(MatchError(testErr))
	})
})
