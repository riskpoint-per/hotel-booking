package service

import (
	"crypto/rand"
	"fmt"
)

const (
	referenceAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	referenceLength   = 8
)

// NewReference returns a random booking reference such as "BK-7K2M9QXA"
func NewReference() (string, error) {
	buf := make([]byte, referenceLength)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate reference: %w", err)
	}

	for i := range buf {
		buf[i] = referenceAlphabet[int(buf[i])%len(referenceAlphabet)]
	}
	return "BK-" + string(buf), nil
}
