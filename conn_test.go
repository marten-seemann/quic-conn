package quicconn

import (
	"context"
	"crypto/tls"
	"errors"
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
	acceptError    error

	streamToOpen quic.Stream
	openError    error

	closed          bool
	closedWithError string
}

func (m *mockSession) AcceptStream(context.Context) (quic.Stream, error) {
	if m.acceptError != nil {
		return nil, m.acceptError
	}
	// AcceptStream blocks until a stream is available
	if m.streamToAccept == nil {
		time.Sleep(time.Hour)
	}
	return m.streamToAccept, nil
}

func (m *mockSession) OpenStream() (quic.Stream, error) {
	if m.openError != nil {
		return nil, m.openError
	}
	return m.streamToOpen, nil
}

func (m *mockSession) OpenStreamSync(context.Context) (quic.Stream, error) {
	return m.streamToOpen, nil
}

func (m *mockSession) LocalAddr() net.Addr {
	return m.localAddr
}

func (m *mockSession) RemoteAddr() net.Addr {
	return m.remoteAddr
}

func (m *mockSession) CloseWithError(_ quic.ErrorCode, e string) error {
	m.closedWithError = e
	m.closed = true
	return nil
}

func (m *mockSession) Close() error {
	return m.CloseWithError(0, "")
}

func (m *mockSession) AcceptUniStream(context.Context) (quic.ReceiveStream, error) {
	panic("not implemented")
}
func (m *mockSession) OpenUniStream() (quic.SendStream, error) { panic("not implemented") }
func (m *mockSession) OpenUniStreamSync(context.Context) (quic.SendStream, error) {
	panic("not implemented")
}
func (m *mockSession) ConnectionState() tls.ConnectionState { panic("not implemented") }
func (m *mockSession) Context() context.Context             { panic("not implemented") }

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

	It("errors when the send stream can't be opened", func() {
		testErr := errors.New("test error")
		sess.openError = testErr
		_, err := newConn(sess)
		Expect(err).To(MatchError(testErr))
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

	It("errors if accepting the stream fails", func() {
		testErr := errors.New("test error")
		sess.acceptError = testErr
		_, err := c.Read(make([]byte, 1))
		Expect(err).To(MatchError(testErr))
	})

	It("immediately closes the receive stream", func() {
		receiveStream.dataToRead.Write([]byte("foobar"))
		sess.streamToAccept = receiveStream

		_, err := c.Read(make([]byte, 1))
		Expect(err).ToNot(HaveOccurred())
		Expect(receiveStream.closed).To(BeTrue())
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

	It("closes", func() {
		c.Close()
		Expect(sess.closed).To(BeTrue())
		Expect(sess.closedWithError).To(BeEmpty())
	})
})
