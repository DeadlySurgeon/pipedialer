package pipedialer

import (
	"context"
	"net"
	"sync"
)

type ConnPool struct {
	connLock  sync.Mutex
	connIndex map[net.Conn]bool
	accept    chan net.Conn

	closed bool
}

func New() *ConnPool {
	return &ConnPool{
		connIndex: map[net.Conn]bool{},
		accept:    make(chan net.Conn),
	}
}

func (rp *ConnPool) Accept() (net.Conn, error) {
	conn := <-rp.accept
	if conn == nil {
		return nil, net.ErrClosed
	}

	return conn, nil
}

func (rp *ConnPool) Close() error {
	rp.connLock.Lock()
	defer rp.connLock.Unlock()

	if rp.closed {
		return nil
	}
	rp.closed = true

	close(rp.accept)

	for conn := range rp.connIndex {
		conn.Close()
		delete(rp.connIndex, conn)
	}

	return nil
}

func (rp *ConnPool) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	rp.connLock.Lock()
	defer rp.connLock.Unlock()

	if rp.closed {
		return nil, net.ErrClosed
	}

	client, server := net.Pipe()

	rp.connIndex[client] = true
	rp.connIndex[server] = true

	rp.accept <- rp.bind(server)

	return rp.bind(client), nil
}

func (rp *ConnPool) Network() string { return "dial" }
func (rp *ConnPool) String() string  { return "connpool" }
func (rp *ConnPool) Addr() net.Addr  { return rp }

func (rp *ConnPool) bind(conn net.Conn) net.Conn {
	return &boundClose{Conn: conn, close: func() {
		rp.connLock.Lock()
		defer rp.connLock.Unlock()

		delete(rp.connIndex, conn)
	}}
}

type boundClose struct {
	net.Conn
	close func()
}

func (bc *boundClose) Close() error {
	bc.close()
	return bc.Conn.Close()
}
