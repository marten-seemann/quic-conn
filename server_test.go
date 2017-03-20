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
	id          protocol.StreamID
	closed      bool
	dataWritten bytes.Buffer
	dataToRead  bytes.Buffer
}

func (m *mockStream) Read(p []byte) (int, error) {
	return m.dataToRead.Read(p)
}

func (m *mockStream) Close() error {
	m.closed = true
	return nil
}

func (m *mockStream) Write(p []byte) (int, error) {
	return m.dataWritten.Write(p)
}

func (m *mockStream) StreamID() protocol.StreamID {
	return m.id
}

func (m *mockStream) Reset(error) {
	panic("not implemented")
}

var _ quic.Stream = &mockStream{}

var _ = Describe("Server", func() {
	var (
		mconn *mockPacketConn
		s     *server
	)

	BeforeEach(func() {
		mconn = &mockPacketConn{}
		s = &server{conn: mconn}
	})

	It("waits for new connections", func() {
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
		var returned bool
		var qconn net.Conn

		go func() {
			defer GinkgoRecover()
			var err error
			qconn, err = s.Accept()
			Expect(err).ToNot(HaveOccurred())
			returned = true
		}()

		sess := &mockSession{}
		s.connStateCallback(sess, quic.ConnStateVersionNegotiated)
		Consistently(func() bool { return returned }).Should(BeFalse())
		s.connStateCallback(sess, quic.ConnStateInitial)
		Consistently(func() bool { return returned }).Should(BeFalse())
		s.connStateCallback(sess, quic.ConnStateForwardSecure)
		Eventually(func() bool { return returned }).Should(BeTrue())
		Expect(qconn).ToNot(BeNil())
		Expect(qconn.(*conn).session).To(Equal(sess))
	})

	It("returns the address of the underlying conn", func() {
		addr := &net.UDPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 1337}
		mconn.addr = addr
		Expect(s.Addr()).To(Equal(addr))
	})

	It("unblocks Accepts when it is closed", func() {
		var returned bool

		// we need to use a real conn here, not the mock conn
		// the mockPacketConn doesn't unblock the ReadFrom when it is closed
		udpAddr, err := net.ResolveUDPAddr("udp", "localhost:12345")
		Expect(err).ToNot(HaveOccurred())
		udpConn, err := net.ListenUDP("udp", udpAddr)
		Expect(err).ToNot(HaveOccurred())
		s.conn = udpConn

		go func() {
			defer GinkgoRecover()
			_, _ = s.Accept()
			returned = true
		}()

		Consistently(func() bool { return returned }).Should(BeFalse())
		err = s.Close()
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool { return returned }).Should(BeTrue())
	})
})