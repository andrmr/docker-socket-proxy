package proxy_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrmr/docker-socket-proxy/pkg/auth"
	"github.com/andrmr/docker-socket-proxy/pkg/proxy"
)

func TestSecurityHandler(t *testing.T) {
	policy := &auth.Policy{
		Groups: map[string][]string{
			"CONTAINERS": {`^/containers/json$`},
		},
	}
	authorizer := auth.NewAuthorizer(policy)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	l := slog.Default()
	p, _ := proxy.NewUnixSocketProxy("/tmp/fake.sock", l)
	handler := &proxy.SecurityHandler{
		Proxy:      p,
		Authorizer: authorizer,
		Logger:     l,
	}

	tests := []struct {
		name   string
		path   string
		method string
		status int
	}{
		{"allowed path", "/containers/json", "GET", http.StatusOK},
		{"blocked path", "/secret", "GET", http.StatusForbidden},
		{"blocked method", "/containers/json", "POST", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tt.status == http.StatusOK {
				return
			}

			if rr.Code != tt.status {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.status)
			}
		})
	}
}
