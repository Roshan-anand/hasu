package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const CSRF_TOKEN_EXPIRY = 1 * 60 * 60 // 1 hour in seconds

// GenerateCSRFToken creates a cryptographically random state token for OAuth redirect flows.
// The token is base64url-encoded to be safe for use as a URL query parameter.
func GenerateCSRFToken() (string, error) {
	bt := make([]byte, 32)
	if _, err := rand.Read(bt); err != nil {
		return "", fmt.Errorf("generate CSRF token error : %w", err)
	}

	return base64.URLEncoding.EncodeToString(bt), nil
}
