package pq

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber768"
)

// KyberKeyPair represents a Kyber key pair
type KyberKeyPair struct {
	PublicKey  kem.PublicKey
	PrivateKey kem.PrivateKey
	Scheme     kem.Scheme
}

// KyberEncapsulation represents a Kyber encapsulation result
type KyberEncapsulation struct {
	Ciphertext []byte
	SharedKey  []byte
}

// GenerateKyberKeyPair generates a new Kyber768 key pair
func GenerateKyberKeyPair() (*KyberKeyPair, error) {
	scheme := kyber768.Scheme()

	publicKey, privateKey, err := scheme.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	return &KyberKeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Scheme:     scheme,
	}, nil
}

// Encapsulate performs Kyber encapsulation
func (kp *KyberKeyPair) Encapsulate() (*KyberEncapsulation, error) {
	ciphertext, sharedSecret, err := kp.Scheme.Encapsulate(kp.PublicKey)
	if err != nil {
		return nil, err
	}

	return &KyberEncapsulation{
		Ciphertext: ciphertext,
		SharedKey:  sharedSecret,
	}, nil
}

// Decapsulate performs Kyber decapsulation
func (kp *KyberKeyPair) Decapsulate(ciphertext []byte) ([]byte, error) {
	if kp.PrivateKey == nil {
		return nil, errors.New("private key not available")
	}

	sharedSecret, err := kp.Scheme.Decapsulate(kp.PrivateKey, ciphertext)
	if err != nil {
		return nil, err
	}

	return sharedSecret, nil
}

// EncapsulateWithPublicKey performs encapsulation with just the public key
func EncapsulateWithPublicKey(publicKey kem.PublicKey) (*KyberEncapsulation, error) {
	scheme := kyber768.Scheme()

	ciphertext, sharedSecret, err := scheme.Encapsulate(publicKey)
	if err != nil {
		return nil, err
	}

	return &KyberEncapsulation{
		Ciphertext: ciphertext,
		SharedKey:  sharedSecret,
	}, nil
}

// DeriveKey derives a symmetric key from the Kyber shared secret
func DeriveKey(sharedSecret []byte, salt []byte) []byte {
	h := sha256.New()
	h.Write(sharedSecret)
	h.Write(salt)
	return h.Sum(nil)
}

// XOREncrypt performs XOR encryption with the derived key
func XOREncrypt(data []byte, key []byte) []byte {
	encrypted := make([]byte, len(data))
	keyLen := len(key)

	for i := 0; i < len(data); i++ {
		encrypted[i] = data[i] ^ key[i%keyLen]
	}

	return encrypted
}

// XORDecrypt performs XOR decryption (same as encryption)
func XORDecrypt(encrypted []byte, key []byte) []byte {
	return XOREncrypt(encrypted, key)
}

// GenerateRandomSalt generates a random salt
func GenerateRandomSalt() ([]byte, error) {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// EncryptMessage encrypts a message using Kyber KEM
func EncryptMessage(message []byte, publicKey kem.PublicKey) ([]byte, []byte, []byte, error) {
	// Encapsulate to get shared secret
	encap, err := EncapsulateWithPublicKey(publicKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// Generate random salt
	salt, err := GenerateRandomSalt()
	if err != nil {
		return nil, nil, nil, err
	}

	// Derive encryption key
	key := DeriveKey(encap.SharedKey, salt)

	// Encrypt message
	ciphertext := XOREncrypt(message, key)

	return ciphertext, encap.Ciphertext, salt, nil
}

// DecryptMessage decrypts a message using Kyber KEM
func DecryptMessage(ciphertext []byte, kemCiphertext []byte, salt []byte, privateKey kem.PrivateKey) ([]byte, error) {
	scheme := kyber768.Scheme()

	// Decapsulate to get shared secret
	sharedSecret, err := scheme.Decapsulate(privateKey, kemCiphertext)
	if err != nil {
		return nil, err
	}

	// Derive decryption key
	key := DeriveKey(sharedSecret, salt)

	// Decrypt message
	message := XORDecrypt(ciphertext, key)

	return message, nil
}

// UnmarshalKyberKeyPair reconstructs a KyberKeyPair from serialized public and private key bytes
func UnmarshalKyberKeyPair(pubKeyBytes, privKeyBytes []byte) (*KyberKeyPair, error) {
	scheme := kyber768.Scheme()

	publicKey, err := scheme.UnmarshalBinaryPublicKey(pubKeyBytes)
	if err != nil {
		return nil, errors.New("failed to unmarshal public key: " + err.Error())
	}

	privateKey, err := scheme.UnmarshalBinaryPrivateKey(privKeyBytes)
	if err != nil {
		return nil, errors.New("failed to unmarshal private key: " + err.Error())
	}

	return &KyberKeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Scheme:     scheme,
	}, nil
}

// DecapsulateWithPrivateKey performs Kyber decapsulation with just a private key
func DecapsulateWithPrivateKey(privateKey kem.PrivateKey, ciphertext []byte) ([]byte, error) {
	scheme := kyber768.Scheme()
	sharedSecret, err := scheme.Decapsulate(privateKey, ciphertext)
	if err != nil {
		return nil, err
	}
	return sharedSecret, nil
}

// BigIntToBytes converts a big.Int to bytes
func BigIntToBytes(n *big.Int) []byte {
	return n.Bytes()
}

// BytesToBigInt converts bytes to a big.Int
func BytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}
