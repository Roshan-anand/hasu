package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

var (
	// TODO : generate dynamically and insert it into env var
	masterSecret string = "N1PCdw3M2B1TfJhoaY2mL736p2vCUc47"
)

// Encrypt GitHub App PEM key using AES-256-GCM
// Uses PBKDF2 to derive encryption key from master secret
func EncryptPEM(plaintext string) (string, error) {
	// Derive 32-byte key from master secret using PBKDF2
	salt := []byte("godploy-github-app")
	key := pbkdf2.Key([]byte(masterSecret), salt, 10000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt GitHub App PEM key using AES-256-GCM
// Returns decrypted PEM key as byte slice
func DecryptPEM(ciphertext string) ([]byte, error) {
	salt := []byte("godploy-github-app")
	key := pbkdf2.Key([]byte(masterSecret), salt, 10000, 32, sha256.New)

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}
