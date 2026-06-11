package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT"}`))

// Claims represents the JWT payload.
type Claims struct {
	Sub string `json:"sub"` // "Name <email>" — UserIdentity.Raw
	Exp int64  `json:"exp"` // Unix timestamp
	Iat int64  `json:"iat"` // Unix timestamp
}

// SignToken creates a signed JWT string from the given claims.
func SignToken(key ed25519.PrivateKey, claims Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)

	message := jwtHeader + "." + payloadEnc
	sig := ed25519.Sign(key, []byte(message))
	sigEnc := base64.RawURLEncoding.EncodeToString(sig)

	return message + "." + sigEnc, nil
}

// VerifyToken parses and verifies a JWT string, returning the claims.
func VerifyToken(pubKey ed25519.PublicKey, tokenStr string) (Claims, error) {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("malformed token")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return Claims{}, fmt.Errorf("parse header: %w", err)
	}
	if header.Alg != "EdDSA" {
		return Claims{}, fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, fmt.Errorf("decode signature: %w", err)
	}

	message := parts[0] + "." + parts[1]
	if !ed25519.Verify(pubKey, []byte(message), sig) {
		return Claims{}, fmt.Errorf("invalid signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("decode payload: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Claims{}, fmt.Errorf("parse claims: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return Claims{}, fmt.Errorf("token expired")
	}

	return claims, nil
}
