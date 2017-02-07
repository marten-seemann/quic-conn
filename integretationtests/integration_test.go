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
	"time"

	quicconn "github.com/marten-seemann/quic-conn"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration tests", func() {
	var data []byte
	var tlsConfig *tls.Config
	const dataLen = (1 << 20) // 1 MB

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
	})

	It("transfers data from the client to the server", func() {
		receivedData := make([]byte, dataLen)
		// start the server
		go func() {
			defer GinkgoRecover()
			ln, err := quicconn.Listen("udp", ":12345", tlsConfig)
			Expect(err).ToNot(HaveOccurred())
			serverConn, err := ln.Accept("localhost:12345")
			Expect(err).ToNot(HaveOccurred())
			// receive data
			_, err = io.ReadFull(serverConn, receivedData)
			Expect(err).ToNot(HaveOccurred())
		}()

		tlsConf := &tls.Config{InsecureSkipVerify: true}
		clientConn, err := quicconn.Dial("localhost:12345", tlsConf)
		Expect(err).ToNot(HaveOccurred())
		// send data
		_, err = clientConn.Write(data)
		Expect(err).ToNot(HaveOccurred())
		// check received data
		Eventually(func() []byte { return receivedData }).Should(Equal(data))
	})
})
