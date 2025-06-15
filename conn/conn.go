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
	"strconv"
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
	c          quic.Connection
	listener   *quic.Listener
	stream     quic.Stream
	LastRecvAt int64
	Stat       int
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

func (q *QuicConn) Accept(newC chan *QuicConn) {
	var addr string
	if q.Addr == "" {
		addr = "0.0.0.0:" + strconv.Itoa(q.Port)
	} else {
		addr = utils.UrlCombine(q.Addr, q.Port, "")
	}

	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	if err != nil {
		ulog.Log().I("quic_accept", "listen error: "+err.Error())
		return
	}
	q.listener = listener

	for {
		c, err := listener.Accept(context.Background())
		if err != nil {
			ulog.Log().I("quic_accept", "accept error: "+err.Error())
			return
		}

		stream, err := c.AcceptStream(context.Background())
		if err != nil {
			ulog.Log().I("quic_accept", "stream error: "+err.Error())
			c.CloseWithError(2, "open stream failed")
			continue
		}

		ulog.Log().I("quic_accept", "new connection from "+c.RemoteAddr().String())
		qConn := &QuicConn{
			BaseConn: BaseConn{
				Addr: c.RemoteAddr().String(),
				Port: 0, // QUIC doesn't have separate ports for connections
			},
			c:      c,
			stream: stream,
		}
		newC <- qConn
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
		NextProtos:         []string{"ucs-quic"},
	}
	c, err := quic.DialAddr(context.Background(), utils.UrlCombine(q.Addr, q.Port, ""), tlcConfig, nil)
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
	if q.stream != nil {
		q.stream.Close()
	}
	if q.c != nil {
		err := q.c.CloseWithError(0, "")
		if err != nil {
			ulog.Log().I("quic_close", "close error: "+err.Error())
			return -1
		}
	}
	if q.listener != nil {
		err := q.listener.Close()
		if err != nil {
			ulog.Log().I("quic_close", "listener close error: "+err.Error())
		}
	}
	return 0
}

func (q *QuicConn) _taskRecv(rx chan []byte) {
	buff := make([]byte, 1024)
	for {
		n, err := q.stream.Read(buff)
		if err != nil {
			// Check if it's a connection close error
			if err.Error() == "Application error 0x0 (remote)" ||
				err.Error() == "NO_ERROR" ||
				err.Error() == "stream canceled" {
				ulog.Log().I("quic_recv", "connection closed gracefully")
				return
			}
			ulog.Log().I("quic_recv", "read error: "+err.Error())
			time.Sleep(1 * time.Millisecond)
			continue
		}
		if n > 0 {
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
			_, err := q.stream.Write(tx_buff)
			if err != nil {
				if err.Error() == "Application error 0x0 (remote)" ||
					err.Error() == "NO_ERROR" ||
					err.Error() == "stream canceled" {
					ulog.Log().I("quic_write", "connection closed, stopping write task")
					return
				}
				ulog.Log().I("quic_write", "write error: "+err.Error())
			}
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
				Addr: c.RemoteAddr().(*net.TCPAddr).IP.String(),
				Port: c.RemoteAddr().(*net.TCPAddr).Port,
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

type UdpConn struct {
	BaseConn
	c          *net.UDPConn
	remoteAddr *net.UDPAddr
	LastRecvAt int64
	Stat       int
	rxChan     chan []byte
}

func NewUdpConn(addr string, port int) *UdpConn {
	return &UdpConn{
		BaseConn: BaseConn{
			Addr: addr,
			Port: port,
		},
	}
}

func (u *UdpConn) Accept(newC chan *UdpConn) {
	var addr net.UDPAddr
	if u.Addr == "" {
		addr = net.UDPAddr{
			IP:   nil, // Listen on all interfaces
			Port: u.Port,
		}
	} else {
		addr = net.UDPAddr{
			IP:   net.ParseIP(u.Addr),
			Port: u.Port,
		}
	}

	l, err := net.ListenUDP("udp", &addr)
	if err != nil {
		ulog.Log().I("udp_accept", "listen error: "+err.Error())
		return
	}
	u.c = l

	// For UDP server, we don't have traditional "connections", but we track clients
	clients := make(map[string]*UdpConn)
	buff := make([]byte, 1024)

	for {
		n, clientAddr, err := l.ReadFromUDP(buff)
		if err != nil {
			// Check if it's a connection close error
			if err.Error() == "use of closed network connection" ||
				err.Error() == "connection closed" {
				ulog.Log().I("udp_accept", "listener closed, stopping accept")
				return
			}
			ulog.Log().I("udp_accept", "read error: "+err.Error())
			continue
		}

		clientKey := clientAddr.String()
		client, exists := clients[clientKey]

		if !exists {
			ulog.Log().I("udp_accept", "new client from "+clientAddr.String())
			client = &UdpConn{
				BaseConn: BaseConn{
					Addr: clientAddr.IP.String(),
					Port: clientAddr.Port,
				},
				c:          l,
				remoteAddr: clientAddr,
			}
			clients[clientKey] = client
			newC <- client
		}

		// Forward the received data to the client's receiver if it has one
		if client.rxChan != nil {
			client.rxChan <- buff[:n]
		}
	}
}

func (u *UdpConn) Connect() int {
	addr := net.UDPAddr{
		IP:   net.ParseIP(u.Addr),
		Port: u.Port,
	}
	c, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		return -1
	}
	u.c = c
	// Don't set remoteAddr for client - this is only for server-side client tracking
	return 0
}

func (u *UdpConn) Close() int {
	if u.c != nil {
		err := u.c.Close()
		if err != nil {
			ulog.Log().I("udp_close", "close error: "+err.Error())
			return -1
		}
	}
	return 0
}

func (u *UdpConn) _taskRecv(rx chan []byte) {
	u.rxChan = rx
	buff := make([]byte, 1024)
	for {
		var n int
		var err error

		if u.remoteAddr != nil {
			// Server mode - this is handled in Accept(), just wait for data
			time.Sleep(1 * time.Millisecond)
			continue
		}

		// Client mode - read directly from connection
		n, err = u.c.Read(buff)

		if err != nil {
			// Check if it's a connection close error
			if err.Error() == "use of closed network connection" ||
				err.Error() == "connection closed" {
				ulog.Log().I("udp_recv", "connection closed, stopping receive")
				return
			}
			ulog.Log().I("udp_recv", "read error: "+err.Error())
			time.Sleep(1 * time.Millisecond)
			return
		} else {
			if n > 0 {
				rx <- buff[:n]
			} else {
				time.Sleep(1 * time.Millisecond)
			}
		}
	}
}

func (u *UdpConn) StartRecv(rx chan []byte) {
	go u._taskRecv(rx)
}

func (u *UdpConn) _task_write(tx chan []byte) {
	for {
		select {
		case tx_buff := <-tx:
			if u.remoteAddr != nil {
				// Server mode - responding to specific client, or client mode
				_, err := u.c.WriteToUDP(tx_buff, u.remoteAddr)
				if err != nil {
					if err.Error() == "use of closed network connection" ||
						err.Error() == "connection closed" {
						ulog.Log().I("udp_write", "connection closed, stopping write task")
						return
					}
					ulog.Log().I("udp_write", "write error: "+err.Error())
				}
			} else {
				// Client mode with DialUDP connection
				_, err := u.c.Write(tx_buff)
				if err != nil {
					if err.Error() == "use of closed network connection" ||
						err.Error() == "connection closed" {
						ulog.Log().I("udp_write", "connection closed, stopping write task")
						return
					}
					ulog.Log().I("udp_write", "write error: "+err.Error())
				}
			}
		}
	}
}

func (u *UdpConn) StartWrite(tx chan []byte) {
	go u._task_write(tx)
}
