package auth

import (
	"testing"
)

func TestJwt(t *testing.T) {
	u := AuthUser{
		Email: "test@test.com",
		Name:  "Test User",
	}

	secret := "testsecretkey"

	var jwtToken string
	var claims *CustomClaims
	var err error

	t.Run("generate JWT", func(t *testing.T) {
		jwtToken, err = generateJWT(u, secret)
		if err != nil {
			t.Fatalf("Failed to generate JWT: %v", err)
		}
	})

	t.Run("verify JWT", func(t *testing.T) {
		claims, err = VerifyJWT(jwtToken, secret)
		if err != nil {
			t.Fatalf("Failed to verify JWT: %v", err)
		}
	})

	t.Run("match JWT", func(t *testing.T) {
		if claims.AuthUser != u {
			t.Errorf("Expected user %v \n got %v", u, claims.AuthUser)
		}
	})

}
