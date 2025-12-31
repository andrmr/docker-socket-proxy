package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	showUsage := func() {
		fmt.Fprintf(os.Stderr, "Docker Socket Proxy using JSON policy for access control.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  docker-socket-proxy [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Configuration can be provided via flags or environment variables.\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "DOCKER_SOCKET_PATH=/run/docker.sock docker-socket-proxy -listen-addr :2376\n")
	}

	policyPath := flag.String("policy", getEnv("POLICY", "policy.json"), "Path to the authorization policy JSON file")
	listenAddr := flag.String("listen-addr", getEnv("LISTEN_ADDR", ":2375"), "Address to listen on")
	socketPath := flag.String(
		"socket-path",
		getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		"Path to the Docker Unix socket",
	)
	flag.Parse()

	logger := initLogger()

	logger.Info("starting docker socket proxy", "listen", *listenAddr, "socket", *socketPath, "policy", *policyPath)

	if _, err := os.Stat(*policyPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Policy file not found: %s\n\n", *policyPath)
		showUsage()
		os.Exit(1)
	}

	pol, err := auth.LoadPolicy(*policyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load policy: %v\n", err)
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
