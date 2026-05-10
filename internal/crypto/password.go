package cryptoutil

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const defaultKey = "qzkj1kjghd=876&*"

func EncryptBUPTPassword(password string) (string, error) {
	return EncryptBUPTPasswordWithKey(password, defaultKey)
}

func EncryptBUPTPasswordWithKey(password, key string) (string, error) {
	plain, err := json.Marshal(password)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("create aes cipher: %w", err)
	}

	padded := pkcs7Pad(plain, block.BlockSize())
	encrypted := make([]byte, len(padded))
	for start := 0; start < len(padded); start += block.BlockSize() {
		block.Encrypt(encrypted[start:start+block.BlockSize()], padded[start:start+block.BlockSize()])
	}

	once := base64.StdEncoding.EncodeToString(encrypted)
	return base64.StdEncoding.EncodeToString([]byte(once)), nil
}

func pkcs7Pad(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	out := make([]byte, len(src)+padding)
	copy(out, src)
	for i := len(src); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}
