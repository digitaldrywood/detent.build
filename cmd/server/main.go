package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"detent.build/internal/config"
	"detent.build/internal/content"
	"detent.build/internal/handler"
	"detent.build/internal/middleware"
	"detent.build/internal/release"

	"github.com/labstack/echo/v4"
)

type echoServer struct {
	e *echo.Echo
}

func (s echoServer) Start(address string) error {
	return s.e.Start(address)
}

func (s echoServer) Shutdown(ctx context.Context) error {
	err := s.e.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return s.e.Server.Shutdown(ctx)
	}
	return err
}

func (s echoServer) Close() error {
	err := s.e.Close()
	if errors.Is(err, http.ErrServerClosed) {
		return s.e.Server.Close()
	}
	return err
}

var _ server = echoServer{}

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// In production the port is not ours to choose: the Dokploy domain entry is
	// bound to one container port, so a process that quietly moves to the next
	// free port is healthy internally and unroutable from outside. Fail instead.
	ln, actualPort, err := listen(cfg.Port, cfg.IsProduction())
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", cfg.Port, err)
	}
	e.Listener = ln
	configureHTTPServer(e.Server)

	if actualPort != cfg.Port {
		slog.Warn("configured port unavailable, using next available", "configured", cfg.Port, "actual", actualPort)
		cfg.Port = actualPort
		cfg.Site.URL = replacePort(cfg.Site.URL, actualPort)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Keep the displayed release current without a redeploy. Off outside
	// production so `make dev` and the tests never depend on the network.
	versions := release.New(content.Version)
	if cfg.IsProduction() {
		versions.Start(signalCtx)
	}

	middleware.Setup(e, cfg, versions)

	h := handler.New(cfg)
	h.RegisterRoutes(e)

	slog.Info("starting server", "url", fmt.Sprintf("http://localhost:%s", cfg.Port), "env", cfg.Env)
	if cfg.TailscaleHostname != "" {
		slog.Info("network access", "url", fmt.Sprintf("http://%s:%s", cfg.TailscaleHostname, cfg.Port))
	}
	if err := serve(signalCtx, echoServer{e: e}, shutdownTimeout); err != nil {
		return err
	}

	slog.Info("server stopped")
	return nil
}
