package quicconn

import (
	"bytes"
	"io"
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockPacketConn struct {
	addr          net.Addr
	dataToRead    []byte
	dataReadFrom  net.Addr
	readErr       error
	dataWritten   bytes.Buffer
	dataWrittenTo net.Addr
	closed        bool
}

func (c *mockPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	if c.readErr != nil {
		return 0, nil, c.readErr
	}
	if c.dataToRead == nil { // block if there's no data
		time.Sleep(time.Hour)
		return 0, nil, io.EOF
	}
	n := copy(b, c.dataToRead)
	c.dataToRead = nil
	return n, c.dataReadFrom, nil
}
func (c *mockPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	c.dataWrittenTo = addr
	return c.dataWritten.Write(b)
}
func (c *mockPacketConn) Close() error                       { c.closed = true; return nil }
func (c *mockPacketConn) LocalAddr() net.Addr                { return c.addr }
func (c *mockPacketConn) SetDeadline(t time.Time) error      { panic("not implemented") }
func (c *mockPacketConn) SetReadDeadline(t time.Time) error  { panic("not implemented") }
func (c *mockPacketConn) SetWriteDeadline(t time.Time) error { panic("not implemented") }

type mockStream struct {
	id     protocol.StreamID
	closed bool
}

func (m *mockStream) Read(p []byte) (int, error) {
	return 0, nil
}

func (m *mockStream) Close() error {
	m.closed = true
	return nil
}

func (m *mockStream) Write(p []byte) (int, error) {
	return 0, nil
}

func (m *mockStream) StreamID() protocol.StreamID {
	return m.id
}

func (m *mockStream) Reset(error) {
	panic("not implemented")
}

var _ quic.Stream = &mockStream{}

type mockSession struct {
	streamToAccept quic.Stream
	streamToOpen   quic.Stream
}

func (m *mockSession) AcceptStream() (quic.Stream, error) {
	return m.streamToAccept, nil
}

func (m *mockSession) OpenStream() (quic.Stream, error) {
	return m.streamToOpen, nil
}

func (m *mockSession) OpenStreamSync() (quic.Stream, error) {
	return m.streamToOpen, nil
}

func (m *mockSession) RemoteAddr() net.Addr {
	return nil
}

func (m *mockSession) Close(error) error {
	return nil
}

var _ quic.Session = &mockSession{}

var _ = Describe("Server", func() {
	var mconn *mockPacketConn

	BeforeEach(func() {
		mconn = &mockPacketConn{}
	})

	It("waits for new connections", func() {
		s := &server{conn: mconn}

		var returned bool
		go func() {
			defer GinkgoRecover()
			_, err := s.Accept()
			Expect(err).ToNot(HaveOccurred())
			returned = true
		}()
		Consistently(func() bool { return returned }).Should(BeFalse())
	})

	It("returns once it has a forward-secure connection", func() {
		s := &server{conn: mconn}
		sess := &mockSession{}

		var returned bool
		var server net.Conn

		go func() {
			defer GinkgoRecover()
			var err error
			server, err = s.Accept()
			Expect(err).ToNot(HaveOccurred())
			returned = true
		}()
		s.connStateCallback(sess, quic.ConnStateVersionNegotiated)
		Consistently(func() bool { return returned }).Should(BeFalse())
		s.connStateCallback(sess, quic.ConnStateInitial)
		Consistently(func() bool { return returned }).Should(BeFalse())
		s.connStateCallback(sess, quic.ConnStateForwardSecure)
		Eventually(func() bool { return returned }).Should(BeTrue())
		Expect(server).To(Equal(s))
	})
})
