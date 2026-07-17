package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// hash the given password
func HashPassword(p string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password error : %w", err)
	}
	return string(hash), nil
}

// check if the given password matches the hashed password
//
// pass : password to check, hash : hashed password
func CheckPasswordHash(pass, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	return err == nil
}
