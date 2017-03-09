package quicconn

import (
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockSession struct {
	remoteAddr net.Addr
	localAddr  net.Addr

	streamToAccept quic.Stream
	streamToOpen   quic.Stream
}

func (m *mockSession) AcceptStream() (quic.Stream, error) {
	// AcceptStream blocks until a stream is available
	if m.streamToAccept == nil {
		time.Sleep(time.Hour)
	}
	return m.streamToAccept, nil
}

func (m *mockSession) OpenStream() (quic.Stream, error) {
	return m.streamToOpen, nil
}

func (m *mockSession) OpenStreamSync() (quic.Stream, error) {
	return m.streamToOpen, nil
}

func (m *mockSession) LocalAddr() net.Addr {
	return m.localAddr
}

func (m *mockSession) RemoteAddr() net.Addr {
	return m.remoteAddr
}

func (m *mockSession) Close(error) error {
	return nil
}

var _ quic.Session = &mockSession{}

var _ = Describe("Conn", func() {
	var (
		c             *conn
		sess          *mockSession
		sendStream    *mockStream
		receiveStream *mockStream
	)

	BeforeEach(func() {
		var err error
		receiveStream = &mockStream{}
		sendStream = &mockStream{}
		sess = &mockSession{
			streamToOpen: sendStream,
		}
		c, err = newConn(sess)
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns the remote address", func() {
		addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 7331}
		sess.remoteAddr = addr
		Expect(c.RemoteAddr()).To(Equal(addr))
	})

	It("returns the local address", func() {
		addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1337}
		sess.localAddr = addr
		Expect(c.LocalAddr()).To(Equal(addr))
	})

	It("writes data", func() {
		n, err := c.Write([]byte("foobar"))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(6))
		Expect(sendStream.dataWritten.Bytes()).To(Equal([]byte("foobar")))
	})

	It("waits with reading until a stream can be accepted", func() {
		var readReturned bool
		go func() {
			defer GinkgoRecover()
			_, err := c.Read(make([]byte, 1))
			Expect(err).ToNot(HaveOccurred())
			readReturned = true
		}()

		Consistently(func() bool { return readReturned }).Should(BeFalse())
	})

	It("reads data", func() {
		receiveStream.dataToRead.Write([]byte("foobar"))
		sess.streamToAccept = receiveStream

		data := make([]byte, 10)
		n, err := c.Read(data)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(6))
		Expect(data).To(ContainSubstring("foobar"))
	})
})
