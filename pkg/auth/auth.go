package auth

import (
	"regexp"
	"strings"
)

var versionRe = regexp.MustCompile(`^/v\d+\.\d+`)

type Authorizer struct {
	allowed []*regexp.Regexp
	denied  []*regexp.Regexp
}

func NewAuthorizer(policy *Policy) *Authorizer {
	auth := &Authorizer{}

	for _, p := range policy.GlobalDeny {
		auth.denied = append(auth.denied, regexp.MustCompile(p))
	}

	for _, patterns := range policy.Groups {
		for _, p := range patterns {
			auth.allowed = append(auth.allowed, regexp.MustCompile(p))
		}
	}

	auth.allowed = append(auth.allowed, regexp.MustCompile(`^/_ping$`))

	return auth
}

func (a *Authorizer) NormalizePath(path string) string {
	path = versionRe.ReplaceAllString(path, "")
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func (a *Authorizer) IsAllowed(path string) bool {
	normalized := a.NormalizePath(path)

	for _, re := range a.denied {
		if re.MatchString(normalized) {
			return false
		}
	}

	for _, re := range a.allowed {
		if re.MatchString(normalized) {
			return true
		}
	}

	return false
}
