package utils

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
)

// BigIntToHex converts big.Int to hex string
func BigIntToHex(i *big.Int) string {
	return hex.EncodeToString(i.Bytes())
}

// HexToBigInt converts hex string to big.Int
func HexToBigInt(s string) (*big.Int, error) {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %v", err)
	}
	return new(big.Int).SetBytes(bytes), nil
}

// BigIntToBase64 converts big.Int to base64 string
func BigIntToBase64(i *big.Int) string {
	return base64.StdEncoding.EncodeToString(i.Bytes())
}

// Base64ToBigInt converts base64 string to big.Int
func Base64ToBigInt(s string) (*big.Int, error) {
	bytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 string: %v", err)
	}
	return new(big.Int).SetBytes(bytes), nil
}

// StructToJSON converts struct to JSON string
func StructToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// JSONToStruct converts JSON string to struct
func JSONToStruct(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

// Int64ToBigInt converts int64 to big.Int
func Int64ToBigInt(n int64) *big.Int {
	return big.NewInt(n)
}

// BigIntToInt64 converts big.Int to int64 (may overflow)
func BigIntToInt64(i *big.Int) int64 {
	return i.Int64()
}

// StringToBytes converts string to byte slice
func StringToBytes(s string) []byte {
	return []byte(s)
}

// BytesToString converts byte slice to string
func BytesToString(b []byte) string {
	return string(b)
}

// HexToBytes converts hex string to bytes
func HexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// BytesToHex converts bytes to hex string
func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// Base64ToBytes converts base64 string to bytes
func Base64ToBytes(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// BytesToBase64 converts bytes to base64 string
func BytesToBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
