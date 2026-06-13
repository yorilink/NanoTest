package auth

import (
	"strconv"
	"strings"
)

type TokenVerifier interface {
	Verify(token string) (int64, error)
}

type DemoTokenVerifier struct{}

func (DemoTokenVerifier) Verify(token string) (int64, error) {
	const prefix = "demo:"
	if !strings.HasPrefix(token, prefix) {
		return 0, ErrInvalidToken
	}
	raw := strings.TrimSpace(strings.TrimPrefix(token, prefix))
	if raw == "" {
		return 0, ErrInvalidToken
	}
	accountID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || accountID < 1 {
		return 0, ErrInvalidToken
	}
	return accountID, nil
}
