package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/andrmr/docker-socket-proxy/pkg/auth"
)

const (
	defaultDialTimeout           = 30 * time.Second
	defaultKeepAlive             = 30 * time.Second
	defaultMaxIdleConns          = 100
	defaultIdleConnTimeout       = 90 * time.Second
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
)

type SecurityHandler struct {
	Proxy      *httputil.ReverseProxy
	Authorizer *auth.Authorizer
	Logger     *slog.Logger
	reqID      uint64
}

func (h *SecurityHandler) nextReqID() string {
	return fmt.Sprintf("%08x", atomic.AddUint64(&h.reqID, 1))
}

func (h *SecurityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := h.nextReqID()

	logger := h.Logger.With(
		"req_id", id,
		"method", r.Method,
		"path", r.URL.Path,
		"remote", r.RemoteAddr,
	)

	logger.Info("request received")

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		logger.Warn("blocked method")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Header.Del("Connection")
	r.Header.Del("Upgrade")
	r.Header.Del("Proxy-Connection")

	if !h.Authorizer.IsAllowed(r.URL.Path) {
		logger.Warn("blocked path")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if h.Authorizer.NormalizePath(r.URL.Path) == "/events" {
		filters := `{"type":{"container":true},"event":{"start":true,"die":true,"destroy":true}}`
		q := r.URL.Query()
		q.Set("filters", filters)
		r.URL.RawQuery = q.Encode()
		logger.Info("events filter injected")
	} else {
		r.URL.RawQuery = ""
	}

	h.Proxy.ServeHTTP(w, r)

	logger.Info("request completed", "duration", time.Since(start).String())
}

func NewUnixSocketProxy(socketPath string, logger *slog.Logger) (*httputil.ReverseProxy, error) {
	target, err := url.Parse("http://docker-unix-socket")
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   defaultDialTimeout,
				KeepAlive: defaultKeepAlive,
			}).DialContext(ctx, "unix", socketPath)
		},
		MaxIdleConns:          defaultMaxIdleConns,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultExpectContinueTimeout,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("proxy error", "method", r.Method, "path", r.URL.String(), "err", err)
		http.Error(w, "Upstream error", http.StatusBadGateway)
	}

	return proxy, nil
}
