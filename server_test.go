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

type mockSession struct {
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

func (m *mockSession) RemoteAddr() net.Addr {
	return nil
}

func (m *mockSession) Close(error) error {
	return nil
}

var _ quic.Session = &mockSession{}

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
		var conn net.Conn

		go func() {
			defer GinkgoRecover()
			var err error
			conn, err = s.Accept()
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
		Expect(conn).To(Equal(s))
	})

	It("returns the address of the underlying conn", func() {
		addr := &net.UDPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 1337}
		mconn.addr = addr
		Expect(s.Addr()).To(Equal(addr))
		Expect(s.LocalAddr()).To(Equal(addr))
	})

	It("writes data", func() {
		var conn net.Conn

		go func() {
			defer GinkgoRecover()
			var err error
			conn, err = s.Accept()
			Expect(err).ToNot(HaveOccurred())
		}()

		time.Sleep(50 * time.Millisecond)
		dataStream := &mockStream{}
		sess := &mockSession{
			streamToOpen: dataStream,
		}
		go s.connStateCallback(sess, quic.ConnStateForwardSecure)
		Eventually(func() net.Conn { return conn }).ShouldNot(BeNil())
		n, err := conn.Write([]byte("foobar"))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(6))
		Expect(dataStream.dataWritten.Bytes()).To(Equal([]byte("foobar")))
	})

	It("waits with reading until a stream can be accepted", func() {
		var conn net.Conn

		go func() {
			defer GinkgoRecover()
			var err error
			conn, err = s.Accept()
			Expect(err).ToNot(HaveOccurred())
		}()

		time.Sleep(50 * time.Millisecond)
		sess := &mockSession{streamToOpen: &mockStream{}}
		s.connStateCallback(sess, quic.ConnStateForwardSecure)
		Eventually(func() net.Conn { return conn }).ShouldNot(BeNil())

		var readReturned bool
		go func() {
			defer GinkgoRecover()
			_, err := s.Read(make([]byte, 1))
			Expect(err).ToNot(HaveOccurred())
			readReturned = true
		}()
		Consistently(func() bool { return readReturned }).Should(BeFalse())
	})

	It("reads data", func() {
		var conn net.Conn

		go func() {
			defer GinkgoRecover()
			var err error
			conn, err = s.Accept()
			Expect(err).ToNot(HaveOccurred())
		}()

		time.Sleep(50 * time.Millisecond)
		dataStream := &mockStream{}
		dataStream.dataToRead.Write([]byte("foobar"))
		sess := &mockSession{
			streamToOpen:   &mockStream{},
			streamToAccept: dataStream,
		}
		s.connStateCallback(sess, quic.ConnStateForwardSecure)
		Eventually(func() net.Conn { return conn }).ShouldNot(BeNil())

		data := make([]byte, 10)
		n, err := conn.Read(data)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(6))
		Expect(data).To(ContainSubstring("foobar"))
	})
})
