package utils

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

// SecureRandomBytes generates n cryptographically secure random bytes
func SecureRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// SecureRandomString generates a random hex string of length n
func SecureRandomString(n int) (string, error) {
	bytes, err := SecureRandomBytes(n / 2)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SecureRandomInt generates a random integer in [0, max)
func SecureRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, nil
	}

	bytes, err := SecureRandomBytes(4)
	if err != nil {
		return 0, err
	}

	// Convert bytes to int
	n := int(bytes[0]) | int(bytes[1])<<8 | int(bytes[2])<<16 | int(bytes[3])<<24
	if n < 0 {
		n = -n
	}

	return n % max, nil
}

// ShuffleBytes shuffles a byte slice using Fisher-Yates algorithm
func ShuffleBytes(data []byte) error {
	n := len(data)
	for i := n - 1; i > 0; i-- {
		j, err := SecureRandomInt(i + 1)
		if err != nil {
			return err
		}
		data[i], data[j] = data[j], data[i]
	}
	return nil
}

// ShuffleInts shuffles an int slice
func ShuffleInts(data []int) error {
	n := len(data)
	for i := n - 1; i > 0; i-- {
		j, err := SecureRandomInt(i + 1)
		if err != nil {
			return err
		}
		data[i], data[j] = data[j], data[i]
	}
	return nil
}
