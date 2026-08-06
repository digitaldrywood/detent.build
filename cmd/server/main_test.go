package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

type stubServer struct {
	started  chan struct{}
	stopped  chan struct{}
	startErr error
	shutdown func(context.Context) error
	closeErr error
	closed   chan struct{}
}

func newStubServer(startErr error) *stubServer {
	return &stubServer{
		started:  make(chan struct{}),
		stopped:  make(chan struct{}),
		startErr: startErr,
		closed:   make(chan struct{}),
	}
}

func (s *stubServer) Start(string) error {
	close(s.started)
	if s.startErr != nil {
		return s.startErr
	}
	<-s.stopped
	return http.ErrServerClosed
}

func (s *stubServer) Shutdown(ctx context.Context) error {
	if s.shutdown != nil {
		return s.shutdown(ctx)
	}
	close(s.stopped)
	return nil
}

func (s *stubServer) Close() error {
	close(s.closed)
	select {
	case <-s.stopped:
	default:
		close(s.stopped)
	}
	return s.closeErr
}

func TestServeGracefulShutdown(t *testing.T) {
	srv := newStubServer(nil)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, srv, time.Second)
	}()

	<-srv.started
	cancel()

	if err := <-result; err != nil {
		t.Fatalf("serve returned an error: %v", err)
	}
}

func TestServeFailure(t *testing.T) {
	wantErr := errors.New("listener failed")
	srv := newStubServer(wantErr)

	err := serve(context.Background(), srv, time.Second)

	if !errors.Is(err, wantErr) {
		t.Fatalf("serve error = %v, want %v", err, wantErr)
	}
}

func TestServeShutdownTimeout(t *testing.T) {
	srv := newStubServer(nil)
	srv.shutdown = func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, srv, time.Millisecond)
	}()

	<-srv.started
	cancel()
	err := <-result

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("serve error = %v, want %v", err, context.DeadlineExceeded)
	}
	select {
	case <-srv.closed:
	default:
		t.Fatal("serve did not force the server closed")
	}
}

func TestServeExpectedServerClosed(t *testing.T) {
	srv := newStubServer(http.ErrServerClosed)

	if err := serve(context.Background(), srv, time.Second); err != nil {
		t.Fatalf("serve returned an error: %v", err)
	}
}

func TestConfigureHTTPServer(t *testing.T) {
	srv := &http.Server{}

	configureHTTPServer(srv)

	if srv.ReadHeaderTimeout != readHeaderTimeout {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, readHeaderTimeout)
	}
	if srv.ReadTimeout != readTimeout {
		t.Errorf("ReadTimeout = %v, want %v", srv.ReadTimeout, readTimeout)
	}
	if srv.WriteTimeout != writeTimeout {
		t.Errorf("WriteTimeout = %v, want %v", srv.WriteTimeout, writeTimeout)
	}
	if srv.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %v, want %v", srv.IdleTimeout, idleTimeout)
	}
}

func TestFindAvailablePortZero(t *testing.T) {
	ln, port, err := findAvailablePort("0")
	if err != nil {
		t.Fatalf("find available port: %v", err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	})

	if port == "0" {
		t.Fatal("findAvailablePort returned port 0")
	}
}

// Production must not silently relocate to another port: the Dokploy domain
// entry is bound to one container port, so a relocated process is healthy
// internally and unreachable from outside.
func TestListenStrictFailsWhenPortIsTaken(t *testing.T) {
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("occupy a port: %v", err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	port := strconv.Itoa(occupied.Addr().(*net.TCPAddr).Port)

	if ln, _, err := listen(port, true); err == nil {
		_ = ln.Close()
		t.Fatal("strict listen succeeded on an occupied port; production would be unroutable")
	}
}

// Development keeps the scan, so several projects can share a machine.
func TestListenNonStrictScansForward(t *testing.T) {
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("occupy a port: %v", err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	port := strconv.Itoa(occupied.Addr().(*net.TCPAddr).Port)

	ln, got, err := listen(port, false)
	if err != nil {
		t.Fatalf("non-strict listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	if got == port {
		t.Errorf("non-strict listen returned the occupied port %s", got)
	}
}
