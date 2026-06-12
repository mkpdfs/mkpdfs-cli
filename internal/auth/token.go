package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// JWTPayload represents the decoded JWT payload fields used by mkp.
type JWTPayload struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Exp               int64  `json:"exp"`
	Iat               int64  `json:"iat"`
}

// DecodeJWT base64-decodes the payload segment of a JWT and unmarshals it.
// It does not verify the signature.
func DecodeJWT(token string) (*JWTPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	payload := parts[1]

	// Add padding if necessary (RawURLEncoding strips it).
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Fallback: some tokens use standard base64.
		decoded, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, err
		}
	}

	var result JWTPayload
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IsTokenExpired returns true if the token is empty, unparseable, or will
// expire within the next 60 seconds (buffer for clock skew / network latency).
func IsTokenExpired(token string) bool {
	if token == "" {
		return true
	}
	payload, err := DecodeJWT(token)
	if err != nil {
		return true
	}
	if payload.Exp == 0 {
		return true
	}
	return time.Unix(payload.Exp, 0).Before(time.Now().Add(60 * time.Second))
}
