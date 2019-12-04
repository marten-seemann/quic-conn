package quicconn

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
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
	id          quic.StreamID
	closed      bool
	dataWritten bytes.Buffer
	dataToRead  bytes.Buffer
}

var _ quic.Stream = &mockStream{}

func (m *mockStream) Read(p []byte) (int, error) {
	return m.dataToRead.Read(p)
}
func (m *mockStream) Close() error {
	m.closed = true
	return nil
}
func (m *mockStream) Write(p []byte) (int, error)      { return m.dataWritten.Write(p) }
func (m *mockStream) StreamID() quic.StreamID          { return m.id }
func (m *mockStream) Context() context.Context         { panic("not implemented") }
func (m *mockStream) SetReadDeadline(time.Time) error  { panic("not implemented") }
func (m *mockStream) SetWriteDeadline(time.Time) error { panic("not implemented") }
func (m *mockStream) SetDeadline(time.Time) error      { panic("not implemented") }
func (m *mockStream) CancelRead(quic.ErrorCode)        { panic("not implemented") }
func (m *mockStream) CancelWrite(quic.ErrorCode)       { panic("not implemented") }

type mockQuicListener struct {
	blockAccept  chan struct{} // close this to make accept return
	sessToAccept *mockSession
	addr         net.Addr
	closeErr     error
	acceptErr    error
}

func newMockQuicListener() *mockQuicListener {
	return &mockQuicListener{
		blockAccept: make(chan struct{}),
	}
}

func (l *mockQuicListener) Accept(context.Context) (quic.Session, error) {
	<-l.blockAccept
	return l.sessToAccept, l.acceptErr
}
func (l *mockQuicListener) Addr() net.Addr { return l.addr }
func (l *mockQuicListener) Close() error   { return l.closeErr }

var _ quic.Listener = &mockQuicListener{}

var _ = Describe("Server", func() {
	var (
		s  *server
		ln *mockQuicListener
	)

	BeforeEach(func() {
		ln = newMockQuicListener()
		s = &server{
			quicServer: ln,
		}
	})

	It("waits for new connections", func() {
		ln.sessToAccept = &mockSession{
			streamToOpen: &mockStream{},
		}
		var returned bool
		go func() {
			defer GinkgoRecover()
			_, err := s.Accept()
			Expect(err).ToNot(HaveOccurred())
			returned = true
		}()
		Consistently(func() bool { return returned }).Should(BeFalse())
		close(ln.blockAccept)
		Eventually(func() bool { return returned }).Should(BeTrue())
	})

	It("errors if it can't accept a connection", func() {
		close(ln.blockAccept)
		testErr := errors.New("accept error")
		ln.acceptErr = testErr
		_, err := s.Accept()
		Expect(err).To(MatchError(testErr))
	})

	It("returns the address of the underlying conn", func() {
		addr := &net.UDPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 1337}
		ln.addr = addr
		Expect(s.Addr()).To(Equal(addr))
	})

	It("closes", func() {
		testErr := errors.New("close error")
		ln.closeErr = testErr
		Expect(s.Close()).To(MatchError(testErr))
	})

	// It("unblocks Accepts when it is closed", func() {
	// 	var returned bool

	// 	// we need to use a real conn here, not the mock conn
	// 	// the mockPacketConn doesn't unblock the ReadFrom when it is closed
	// 	udpAddr, err := net.ResolveUDPAddr("udp", "localhost:12345")
	// 	Expect(err).ToNot(HaveOccurred())
	// 	udpConn, err := net.ListenUDP("udp", udpAddr)
	// 	Expect(err).ToNot(HaveOccurred())
	// 	s.conn = udpConn

	// 	go func() {
	// 		defer GinkgoRecover()
	// 		_, _ = s.Accept()
	// 		returned = true
	// 	}()

	// 	Consistently(func() bool { return returned }).Should(BeFalse())
	// 	err = s.Close()
	// 	Expect(err).ToNot(HaveOccurred())
	// 	Eventually(func() bool { return returned }).Should(BeTrue())
	// })
})
