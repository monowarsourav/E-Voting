package utils

import (
	"crypto/rand"
	"errors"
	"math/big"
)

// RandomBigInt generates a random big integer in [0, max)
func RandomBigInt(max *big.Int) (*big.Int, error) {
	return rand.Int(rand.Reader, max)
}

// RandomBigIntRange generates a random big integer in [min, max)
func RandomBigIntRange(min, max *big.Int) (*big.Int, error) {
	if min.Cmp(max) >= 0 {
		return nil, errors.New("min must be less than max")
	}

	// range = max - min
	rangeVal := new(big.Int).Sub(max, min)

	// random value in [0, range)
	r, err := rand.Int(rand.Reader, rangeVal)
	if err != nil {
		return nil, err
	}

	// result = min + r
	result := new(big.Int).Add(min, r)
	return result, nil
}

// ModInverse computes the modular inverse of a modulo m
func ModInverse(a, m *big.Int) (*big.Int, error) {
	inv := new(big.Int).ModInverse(a, m)
	if inv == nil {
		return nil, errors.New("modular inverse does not exist")
	}
	return inv, nil
}

// LCM computes the least common multiple of a and b
func LCM(a, b *big.Int) *big.Int {
	gcd := new(big.Int).GCD(nil, nil, a, b)
	ab := new(big.Int).Mul(a, b)
	return new(big.Int).Div(ab, gcd)
}

// GCD computes the greatest common divisor of a and b
func GCD(a, b *big.Int) *big.Int {
	return new(big.Int).GCD(nil, nil, a, b)
}

// IsPrime checks if n is probably prime
func IsPrime(n *big.Int, certainty int) bool {
	return n.ProbablyPrime(certainty)
}

// GenerateSafePrime generates a safe prime p = 2q + 1 where q is also prime
func GenerateSafePrime(bits int) (*big.Int, *big.Int, error) {
	for i := 0; i < 1000; i++ {
		q, err := rand.Prime(rand.Reader, bits-1)
		if err != nil {
			return nil, nil, err
		}

		p := new(big.Int).Mul(q, big.NewInt(2))
		p.Add(p, big.NewInt(1))

		if p.ProbablyPrime(20) {
			return p, q, nil
		}
	}

	return nil, nil, errors.New("failed to generate safe prime")
}

// BytesToBigInt converts bytes to big.Int
func BytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

// BigIntToBytes converts big.Int to bytes with fixed length
func BigIntToBytes(i *big.Int, length int) []byte {
	bytes := i.Bytes()
	if len(bytes) >= length {
		return bytes
	}

	// Pad with zeros
	padded := make([]byte, length)
	copy(padded[length-len(bytes):], bytes)
	return padded
}
