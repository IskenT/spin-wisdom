package tcp

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConn реализует интерфейс net.Conn для тестирования
type mockConn struct {
	closed bool
	mu     sync.Mutex
}

func newMockConn() *mockConn {
	return &mockConn{}
}

func (m *mockConn) Read(b []byte) (n int, err error)  { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type mockListener struct {
	conns  chan net.Conn
	closed bool
	mu     sync.Mutex
}

func newMockListener() *mockListener {
	return &mockListener{
		conns: make(chan net.Conn, 1),
	}
}

func (l *mockListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	closed := l.closed
	l.mu.Unlock()

	if closed {
		return nil, errors.New("listener is closed")
	}

	conn, ok := <-l.conns
	if !ok {
		return nil, errors.New("listener is closed")
	}
	return conn, nil
}

func (l *mockListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		close(l.conns)
	}
	return nil
}

func (l *mockListener) Addr() net.Addr {
	return &net.TCPAddr{}
}

func TestNewServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewMockTCPHandler(ctrl)

	tests := []struct {
		name    string
		port    int
		opts    []ServerOption
		wantMax int
	}{
		{
			name: "default_settings",
			port: 8083,
		},
		{
			name:    "with_max_connections",
			port:    8083,
			opts:    []ServerOption{WithMaxConnections(10)},
			wantMax: 10,
		},
		{
			name: "with_timeouts",
			port: 8083,
			opts: []ServerOption{WithTimeouts(5*time.Second, 5*time.Second)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.port, handler, tt.opts...)
			assert.NotNil(t, server)
			assert.Equal(t, tt.port, server.port)
			assert.Equal(t, tt.wantMax, server.maxConns)
		})
	}
}

func TestServer_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewMockTCPHandler(ctrl)
	server := NewServer(8083, handler)

	listener := newMockListener()
	server.listener = listener

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	cancel()

	err := <-errCh
	require.NoError(t, err)
}

func TestServer_HandleConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewMockTCPHandler(ctrl)
	server := NewServer(8083, handler, WithMaxConnections(1))

	conn := newMockConn()

	var wg sync.WaitGroup
	wg.Add(1)

	handler.EXPECT().
		Handle(gomock.Any(), conn).
		Do(func(ctx context.Context, conn net.Conn) {
			wg.Done()
		}).
		Times(1)

	ctx := context.Background()
	server.handleConnection(ctx, conn)

	wg.Wait()

	assert.True(t, conn.closed)
}

func TestServer_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewMockTCPHandler(ctrl)
	server := NewServer(8083, handler)

	listener := newMockListener()
	server.listener = listener

	ctx := context.Background()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	err := server.Shutdown(context.Background())
	require.NoError(t, err)

	err = <-serverErr
	require.NoError(t, err)
}

func TestServer_MaxConnections(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewMockTCPHandler(ctrl)
	maxConns := 100

	server := &Server{
		port:     8080,
		handler:  handler,
		maxConns: maxConns,
		connPool: make(chan net.Conn, maxConns),
	}

	var wg sync.WaitGroup
	wg.Add(maxConns)

	handler.EXPECT().
		Handle(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, conn net.Conn) {
			defer wg.Done()
		}).
		Times(maxConns)

	ctx := context.Background()

	for i := 0; i < maxConns; i++ {
		conn := newMockConn()
		server.connPool <- conn
		go server.handleConnection(ctx, conn)
	}

	wg.Wait()

	extraConn := newMockConn()

	select {
	case server.connPool <- extraConn:
		t.Error("Should not be able to add more connections")
	default:
		extraConn.Close()
	}

	assert.True(t, extraConn.closed)
	assert.Equal(t, maxConns, len(server.connPool), "Should have maximum connections")
}
