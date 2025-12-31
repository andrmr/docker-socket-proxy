package auth

import (
	"encoding/json"
	"os"
)

type Policy struct {
	Groups     map[string][]string `json:"groups"`
	GlobalDeny []string            `json:"global_deny"`
}

func LoadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Policy
	err = json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
