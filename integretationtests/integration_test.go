package integrationtests

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	mrand "math/rand"
	"strconv"
	"time"

	quicconn "github.com/marten-seemann/quic-conn"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration tests", func() {
	var data []byte
	var tlsConfig *tls.Config
	const dataLen = 100 * (1 << 10) // 100 kb
	var port string

	generateTLSConfig := func() {
		key, err := rsa.GenerateKey(rand.Reader, 1024)
		Expect(err).ToNot(HaveOccurred())
		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(time.Hour),
		}
		certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
		Expect(err).ToNot(HaveOccurred())
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

		tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
		Expect(err).ToNot(HaveOccurred())
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{tlsCert}}
	}

	BeforeEach(func() {
		r := mrand.New(mrand.NewSource(int64(time.Now().Nanosecond())))
		data = make([]byte, dataLen)
		_, err := r.Read(data)
		Expect(err).ToNot(HaveOccurred())
		generateTLSConfig()

		port = strconv.Itoa(int(10000 + r.Int31n(40000))) // random port number between 10000 and 50000
	})

	It("transfers data from the client to the server", func(done Done) {
		receivedData := make([]byte, dataLen)
		// start the server
		go func() {
			defer GinkgoRecover()
			ln, err := quicconn.Listen("udp", ":"+port, tlsConfig)
			Expect(err).ToNot(HaveOccurred())
			serverConn, err := ln.Accept()
			Expect(err).ToNot(HaveOccurred())
			// receive data
			_, err = io.ReadFull(serverConn, receivedData)
			Expect(err).ToNot(HaveOccurred())
		}()

		tlsConf := &tls.Config{InsecureSkipVerify: true}
		clientConn, err := quicconn.Dial("localhost:"+port, tlsConf)
		Expect(err).ToNot(HaveOccurred())
		// send data
		_, err = clientConn.Write(data)
		Expect(err).ToNot(HaveOccurred())
		// check received data
		Eventually(func() []byte { return receivedData }).Should(Equal(data))
		close(done)
	}, 10)

	It("transfers data from the client to the server and back", func(done Done) {
		// start the server
		go func() {
			defer GinkgoRecover()
			ln, err := quicconn.Listen("udp", ":"+port, tlsConfig)
			Expect(err).ToNot(HaveOccurred())
			serverConn, err := ln.Accept()
			Expect(err).ToNot(HaveOccurred())
			// receive data
			d := make([]byte, dataLen)
			_, err = io.ReadFull(serverConn, d)
			Expect(err).ToNot(HaveOccurred())
			_, err = serverConn.Write(d)
			Expect(err).ToNot(HaveOccurred())
		}()

		tlsConf := &tls.Config{InsecureSkipVerify: true}
		clientConn, err := quicconn.Dial("localhost:"+port, tlsConf)
		Expect(err).ToNot(HaveOccurred())
		// send data
		_, err = clientConn.Write(data)
		Expect(err).ToNot(HaveOccurred())
		// check received data
		receivedData := make([]byte, dataLen)
		_, err = io.ReadFull(clientConn, receivedData)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() []byte { return receivedData }).Should(Equal(data))
		close(done)
	}, 10)
})
