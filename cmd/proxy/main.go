package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrmr/docker-socket-proxy/pkg/auth"
	"github.com/andrmr/docker-socket-proxy/pkg/proxy"
)

const (
	serverReadTimeout       = 15 * time.Second
	serverReadHeaderTimeout = 10 * time.Second
	serverWriteTimeout      = 30 * time.Second
	serverIdleTimeout       = 120 * time.Second
	shutdownTimeout         = 10 * time.Second
)

func main() {
	policyPath := flag.String("policy", "policy.json", "Path to the authorization policy JSON file")
	listenAddr := flag.String("listen-addr", ":2375", "Address to listen on")
	socketPath := flag.String("socket-path", "/var/run/docker.sock", "Path to the Docker Unix socket")
	flag.Parse()

	logger := initLogger()

	logger.Info("starting docker socket proxy", "listen", *listenAddr, "socket", *socketPath, "policy", *policyPath)

	pol, err := auth.LoadPolicy(*policyPath)
	if err != nil {
		logger.Error("failed to load policy", "err", err)
		os.Exit(1)
	}

	authorizer := auth.NewAuthorizer(pol)

	p, err := proxy.NewUnixSocketProxy(*socketPath, logger)
	if err != nil {
		logger.Error("failed to create proxy", "err", err)
		os.Exit(1)
	}

	handler := &proxy.SecurityHandler{
		Proxy:      p,
		Authorizer: authorizer,
		Logger:     logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadTimeout:       serverReadTimeout,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Warn("shutdown signal received")

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err = server.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", "err", err)
		}
		close(idleConnsClosed)
	}()

	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}

	<-idleConnsClosed
	logger.Info("shutdown complete")
}

func initLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))
}
