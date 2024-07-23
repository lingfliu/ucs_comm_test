package conn

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"lingfliu.github.com/ucs_comm_test/ulog"
	"lingfliu.github.com/ucs_comm_test/utils"
)

type ConnOp interface {
	Connect() error
	Close() error
	StartRecv() error
	StartWrite() error
	InstantWrite([]byte) error
	ScheduleWrite([]byte) error
}

type BaseConn struct {
	Addr string
	Port int
}
type QuicConn struct {
	BaseConn
	c        quic.Connection
	listener quic.Listener
	stream   quic.Stream
}

func NewQuicConn(addr string, port int) *QuicConn {
	return &QuicConn{
		BaseConn: BaseConn{
			Addr: addr,
			Port: port,
		},
	}
}

// copied from the quic-go example
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := func() []byte {
		var buf bytes.Buffer
		if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
			return nil
		}
		return buf.Bytes()
	}()

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"ucs-quic"},
	}
}

func (q *QuicConn) Listen() int {
	listener, err := quic.ListenAddr(utils.UrlCombine(q.Addr, q.Port, ""), generateTLSConfig(), nil)
	if err != nil {
		return -1
	}

	for range time.Tick(1 * time.Millisecond) {
		c, err := listener.Accept(context.Background())
		if err != nil {
			return -1
		}

		stream, err := c.AcceptStream(context.Background())
		if err != nil {
			c.CloseWithError(2, "open stream failed")
			continue
		}
		q.stream = stream
		q.c = c

		break
	}
	return 0
}

func (q *QuicConn) Connect() int {
	tlcConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	c, err := quic.DialAddr(context.Background(), q.Addr, tlcConfig, nil)
	if err != nil {
		return -1
	}

	stream, err := c.OpenStreamSync(context.Background())
	if err != nil {
		return -1
	}
	q.c = c
	q.stream = stream
	return 0
}

func (q *QuicConn) Close() int {
	q.stream.Close()
	err := q.c.CloseWithError(0, "")
	if err != nil {
		return -1
	}
	return 0
}

func (q *QuicConn) _taskRecv(rx chan []byte) {
	buff := make([]byte, 1024)
	for {
		n, err := q.stream.Read(buff)
		if err != nil {
			ulog.Log().Tag("conn").I("read error: ", err)
		}
		if n > 0 {
			// do something
			rx <- buff[:n]
		} else {
			time.Sleep(1 * time.Millisecond)
		}
	}
}

func (q *QuicConn) StartRecv(rx chan []byte) {
	go q._taskRecv(rx)
}

func (q *QuicConn) StartWrite(tx chan []byte) {
	go q._task_write(tx)
}

func (q *QuicConn) _task_write(tx chan []byte) {
	for {
		select {
		case tx_buff := <-tx:
			q.stream.Write(tx_buff)
		}
	}
}

type TcpConn struct {
	BaseConn
	c          *net.TCPConn
	LastRecvAt int64
	Stat       int
}

func NewTcpConn(addr string, port int) *TcpConn {
	return &TcpConn{
		BaseConn: BaseConn{
			Addr: addr,
			Port: port,
		},
	}
}

func (t *TcpConn) Accept(newC chan *TcpConn) {
	addr := net.TCPAddr{
		IP:   net.ParseIP(t.Addr),
		Port: t.Port,
	}
	l, err := net.ListenTCP("tcp", &addr)
	if err != nil {
		ulog.Log().I("accept", "listen error")
		return
	}
	for {
		c, err := l.AcceptTCP()
		if err != nil {
			ulog.Log().I("accept", "accept error")
			return
		}

		ulog.Log().I("accept", "new conn from "+c.RemoteAddr().String())
		t := &TcpConn{
			BaseConn: BaseConn{
				Addr: t.Addr,
				Port: t.Port,
			},
			c: c,
		}
		newC <- t
	}
}

func (t *TcpConn) Connect() int {
	addr := net.TCPAddr{
		IP:   net.ParseIP(t.Addr),
		Port: t.Port,
	}
	c, err := net.DialTCP("tcp", nil, &addr)
	if err != nil {
		return -1
	}
	t.c = c
	return 0
}

func (t *TcpConn) Close() int {
	t.c.Close()
	return 0
}

func (t *TcpConn) _taskRecv(rx chan []byte) {
	buff := make([]byte, 1024)
	for {
		n, err := t.c.Read(buff)
		if err != nil {
			// ulog.Log().Tag("conn").I("read error: ", err)
			time.Sleep(1 * time.Millisecond)
		} else {
			if n > 0 {
				// do something
				rx <- buff[:n]
			} else {
				time.Sleep(1 * time.Millisecond)
			}
		}
	}
}

func (t *TcpConn) StartRecv(rx chan []byte) {
	go t._taskRecv(rx)
}

func (t *TcpConn) _task_write(tx chan []byte) {
	for {
		select {
		case tx_buff := <-tx:
			t.c.Write(tx_buff)
		}
	}
}

func (t *TcpConn) StartWrite(tx chan []byte) {
	go t._task_write(tx)
}
