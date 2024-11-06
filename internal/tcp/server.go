package tcp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"
)

type TCPHandler interface {
	Handle(ctx context.Context, conn net.Conn)
}

type Server struct {
	port     int
	handler  TCPHandler
	listener net.Listener

	activeConns sync.WaitGroup
	connPool    chan net.Conn

	maxConns     int
	readTimeout  time.Duration
	writeTimeout time.Duration

	mu     sync.RWMutex
	closed bool

	shutdownCh chan struct{}
}

type ServerOption func(*Server)

func WithMaxConnections(n int) ServerOption {
	return func(s *Server) {
		s.maxConns = n
	}
}

func WithTimeouts(read, write time.Duration) ServerOption {
	return func(s *Server) {
		s.readTimeout = read
		s.writeTimeout = write
	}
}

func NewServer(port int, handler TCPHandler, opts ...ServerOption) *Server {
	s := &Server{
		port:       port,
		handler:    handler,
		connPool:   make(chan net.Conn, 100),
		shutdownCh: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener

	slog.Info("Server started", "port", s.port, "maxConnections", s.maxConns)

	go s.acceptConnections(ctx)

	for {
		select {
		case <-ctx.Done():
			return s.shutdown()
		case conn, ok := <-s.connPool:
			if !ok {
				return nil
			}
			s.handleConnection(ctx, conn)
		}
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	close(s.shutdownCh)

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			slog.Warn("Error closing listener", "error", err)
		}
	}

	done := make(chan struct{})
	go func() {
		s.activeConns.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown context cancelled: %w", ctx.Err())
	case <-done:
		slog.Info("All connections closed successfully")
		return nil
	}
}

func (s *Server) acceptConnections(ctx context.Context) {
	defer close(s.connPool)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.RLock()
			closed := s.closed
			s.mu.RUnlock()

			if closed {
				return
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				slog.Warn("Temporary error accepting connection", "error", err)
				time.Sleep(time.Second)
				continue
			}

			slog.Error("Error accepting connection", "error", err)
			return
		}

		select {
		case <-ctx.Done():
			conn.Close()
			return
		default:
			if len(s.connPool) < s.maxConns {
				s.connPool <- conn
			} else {
				conn.Close()
				slog.Warn("Connection rejected: max connections reached")
			}
		}
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	s.activeConns.Add(1)
	go func() {
		defer s.activeConns.Done()
		defer conn.Close()

		if s.readTimeout > 0 || s.writeTimeout > 0 {
			conn = &timeoutConn{
				Conn:         conn,
				readTimeout:  s.readTimeout,
				writeTimeout: s.writeTimeout,
			}
		}

		remoteAddr := conn.RemoteAddr().String()
		slog.Info("New connection established",
			"remote", remoteAddr,
			"activeConnections", s.activeConnections(),
		)

		s.handler.Handle(ctx, conn)

		slog.Debug("Connection closed", "remote", remoteAddr)
	}()
}

func (s *Server) shutdown() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	if err := s.listener.Close(); err != nil {
		slog.Warn("Error closing listener", "error", err)
	}

	done := make(chan struct{})
	go func() {
		s.activeConns.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Server shutdown completed")
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("shutdown timeout: some connections didn't close gracefully")
	}
}

func (s *Server) activeConnections() int {
	return len(s.connPool)
}

type timeoutConn struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (c *timeoutConn) Read(b []byte) (n int, err error) {
	if c.readTimeout > 0 {
		err = c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		if err != nil {
			return 0, err
		}
	}
	return c.Conn.Read(b)
}

func (c *timeoutConn) Write(b []byte) (n int, err error) {
	if c.writeTimeout > 0 {
		err = c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
		if err != nil {
			return 0, err
		}
	}
	return c.Conn.Write(b)
}
