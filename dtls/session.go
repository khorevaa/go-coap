package dtls

import (
	"context"
	"fmt"
	"net"

	coapNet "github.com/go-ocf/go-coap/v2/net"
	"github.com/go-ocf/go-coap/v2/udp/client"
	"github.com/go-ocf/go-coap/v2/udp/message/pool"
)

type EventFunc = func()

type Session struct {
	connection     *coapNet.Conn
	maxMessageSize int

	onClose []EventFunc
	onRun   []EventFunc

	cancel context.CancelFunc
	ctx    context.Context
}

func NewSession(
	ctx context.Context,
	connection *coapNet.Conn,
	maxMessageSize int,
) *Session {
	ctx, cancel := context.WithCancel(ctx)
	return &Session{
		ctx:            ctx,
		cancel:         cancel,
		connection:     connection,
		maxMessageSize: maxMessageSize,
	}
}

func (s *Session) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *Session) AddOnClose(f EventFunc) {
	s.onClose = append(s.onClose, f)
}

func (s *Session) AddOnRun(f EventFunc) {
	s.onRun = append(s.onRun, f)
}

func (s *Session) Close() error {
	s.cancel()
	return nil
}

func (s *Session) Context() context.Context {
	return s.ctx
}

func (s *Session) WriteMessage(req *pool.Message) error {
	data, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}
	return s.connection.WriteWithContext(req.Context(), data)
}

func (s *Session) MaxMessageSize() int {
	return s.maxMessageSize
}

func (s *Session) RemoteAddr() net.Addr {
	return s.connection.RemoteAddr()
}

// Run reads and process requests from a connection, until the connection is not closed.
func (s *Session) Run(cc *client.ClientConn) (err error) {
	defer func() {
		err1 := s.Close()
		if err == nil {
			err = err1
		}
		for _, f := range s.onClose {
			f()
		}
	}()
	for _, f := range s.onRun {
		f()
	}
	m := make([]byte, s.maxMessageSize)
	for {
		readBuf := m
		readLen, err := s.connection.ReadWithContext(s.ctx, readBuf)
		if err != nil {
			return err
		}
		readBuf = readBuf[:readLen]
		err = cc.Process(readBuf)
		if err != nil {
			return err
		}
	}
}