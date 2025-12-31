package auth_test

import (
	"testing"

	"github.com/andrmr/docker-socket-proxy/pkg/auth"
)

func TestAuthorizer(t *testing.T) {
	policy := &auth.Policy{
		Groups: map[string][]string{
			"CONTAINERS": {
				`^/containers/json$`,
			},
		},
		GlobalDeny: []string{
			`^/containers/[a-f0-9]{64}/attach`,
		},
	}

	tests := []struct {
		name   string
		policy *auth.Policy
		path   string
		want   bool
	}{
		{
			name:   "ping is always allowed",
			policy: policy,
			path:   "/_ping",
			want:   true,
		},
		{
			name:   "allowed by group",
			policy: policy,
			path:   "/containers/json",
			want:   true,
		},
		{
			name: "blocked if not in any group",
			policy: &auth.Policy{
				Groups:     map[string][]string{},
				GlobalDeny: []string{},
			},
			path: "/containers/json",
			want: false,
		},
		{
			name:   "normalization works",
			policy: policy,
			path:   "/v1.41/containers/json/",
			want:   true,
		},
		{
			name:   "global deny override",
			policy: policy,
			path:   "/containers/1234567890123456789012345678901234567890123456789012345678901234/attach",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := auth.NewAuthorizer(tt.policy)
			if got := a.IsAllowed(tt.path); got != tt.want {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
