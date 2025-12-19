package deduplication

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type Hasher struct {
	algorithm string
}

func NewHasher(algorithm string) *Hasher {
	return &Hasher{algorithm: algorithm}
}

func (h *Hasher) ComputeHash(msg map[string]interface{}, fields []string) (string, error) {
	if len(fields) == 0 {
		return "", fmt.Errorf("no fields specified for hashing")
	}

	var builder strings.Builder

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
		sum := md5.Sum([]byte(input))
		return hex.EncodeToString(sum[:]), nil
	}
}
