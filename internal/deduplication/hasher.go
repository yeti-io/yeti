package deduplication

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Hasher handles message hashing logic
type Hasher struct {
	algorithm string
}

// NewHasher creates a new hasher instance
func NewHasher(algorithm string) *Hasher {
	return &Hasher{algorithm: algorithm}
}

// ComputeHash computes the hash of the message based on specific fields
func (h *Hasher) ComputeHash(msg map[string]interface{}, fields []string) (string, error) {
	if len(fields) == 0 {
		return "", fmt.Errorf("no fields specified for hashing")
	}

	// Create a deterministic string representation
	var builder strings.Builder

	// Iterate ordered by slice
	for _, field := range fields {
		val, exists := msg[field]
		if !exists {
			val = ""
		}
		builder.WriteString(fmt.Sprintf("%v|", val))
	}

	input := builder.String()

	switch h.algorithm {
	case "sha256":
		sum := sha256.Sum256([]byte(input))
		return hex.EncodeToString(sum[:]), nil
	case "md5":
		sum := md5.Sum([]byte(input))
		return hex.EncodeToString(sum[:]), nil
	default:
		// Fallback to md5
		sum := md5.Sum([]byte(input))
		return hex.EncodeToString(sum[:]), nil
	}
}
