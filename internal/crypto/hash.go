package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// SHA3Hash computes SHA3-256 hash of data
func SHA3Hash(data []byte) []byte {
	hasher := sha3.New256()
	hasher.Write(data)
	return hasher.Sum(nil)
}

// SHA3HashMultiple computes SHA3-256 hash of multiple data chunks
func SHA3HashMultiple(data ...[]byte) []byte {
	hasher := sha3.New256()
	for _, d := range data {
		hasher.Write(d)
	}
	return hasher.Sum(nil)
}

// SHA3HashBigInt computes SHA3-256 hash of big integers
func SHA3HashBigInt(values ...*big.Int) []byte {
	hasher := sha3.New256()
	for _, v := range values {
		hasher.Write(v.Bytes())
	}
	return hasher.Sum(nil)
}

// SHA3HashString computes SHA3-256 hash of string
func SHA3HashString(s string) []byte {
	return SHA3Hash([]byte(s))
}

// HashToBigInt converts hash to big integer
func HashToBigInt(hash []byte) *big.Int {
	return new(big.Int).SetBytes(hash)
}

// HashChallenge generates challenge for ZK proofs
// c = Hash(inputs...) mod q
func HashChallenge(q *big.Int, values ...*big.Int) *big.Int {
	hash := SHA3HashBigInt(values...)
	c := new(big.Int).SetBytes(hash)
	c.Mod(c, q)
	return c
}

// HMACSHA256 computes HMAC-SHA256
func HMACSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// VerifyHMAC verifies HMAC
func VerifyHMAC(key, data, expectedMAC []byte) bool {
	mac := HMACSHA256(key, data)
	return hmac.Equal(mac, expectedMAC)
}

// FingerprintToHash converts fingerprint data to secure hash
func FingerprintToHash(fingerprintData []byte) []byte {
	return SHA3Hash(fingerprintData)
}

// DeriveKey derives a key from password using SHA3
func DeriveKey(password, salt []byte) []byte {
	return SHA3HashMultiple(password, salt)
}

// HashToField hashes data and reduces to field element
func HashToField(data []byte, field *big.Int) *big.Int {
	hash := SHA3Hash(data)
	h := new(big.Int).SetBytes(hash)
	h.Mod(h, field)
	return h
}
