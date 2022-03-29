package netplus

import (
	"net"
)

// CounterConn counts all bytes that go through it
type CounterConn struct {
	net.Conn
	Upstream   int64
	Downstream int64
}

// CounterListener is the Listener that uses CounterConn instead of net.Conn
type CounterListener struct {
	net.Listener
}

// Accept wraps the inner net.Listener accept and returns a CounterConn
func (cl CounterListener) Accept() (net.Conn, error) {
	conn, err := cl.Listener.Accept()
	return &CounterConn{conn, 0, 0}, err
}

func (cc *CounterConn) Read(b []byte) (int, error) {
	n, err := cc.Conn.Read(b)
	cc.Upstream += int64(n)
	return n, err
}

func (cc *CounterConn) Write(b []byte) (int, error) {
	n, err := cc.Conn.Write(b)
	cc.Downstream += int64(n)
	return n, err
}
