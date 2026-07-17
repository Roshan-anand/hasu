package security

import (
	"math/rand"
	"strings"

	"github.com/google/uuid"
)

// GenerateRandomID generates a random string of the specified length using alphanumeric characters.
func GenerateRandomID(length int, isLower bool) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range length {
		result[i] = chars[rand.Intn(len(chars))]
	}

	if isLower {
		return strings.ToLower(string(result))
	}

	return string(result)
}

// generates new primary key
func GeneratePrimaryKey() uuid.UUID {
	return uuid.New()
}
