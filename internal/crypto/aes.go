// Package crypto provides AES-128-ECB encryption and decryption for Engine DJ stems.
//
// The encryption key is extracted dynamically at runtime from the local installed
// Engine DJ application binary. No key data is hardcoded or published in this package.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	cachedKeyMu sync.RWMutex
	cachedKey   []byte
)

// Candidate default Engine DJ binary locations.
var engineBinaryPaths = []string{
	"/Applications/Engine DJ.app/Contents/MacOS/OfflineAnalyzer",
	"/Applications/Engine DJ.app/Contents/MacOS/Engine DJ",
}

// DeriveKey dynamically extracts the AES key by scanning installed local Engine DJ binaries.
// Caches the key in package memory after the first successful extraction.
func DeriveKey() ([]byte, error) {
	cachedKeyMu.RLock()
	if len(cachedKey) > 0 {
		keyCopy := make([]byte, len(cachedKey))
		copy(keyCopy, cachedKey)
		cachedKeyMu.RUnlock()
		return keyCopy, nil
	}
	cachedKeyMu.RUnlock()

	cachedKeyMu.Lock()
	defer cachedKeyMu.Unlock()

	// Double-check after acquiring write lock
	if len(cachedKey) > 0 {
		keyCopy := make([]byte, len(cachedKey))
		copy(keyCopy, cachedKey)
		return keyCopy, nil
	}

	// Check custom environment variable override if provided
	if customPath := os.Getenv("ENGINE_DJ_APP_PATH"); customPath != "" {
		if key, err := scanBinaryForKey(customPath); err == nil {
			cachedKey = make([]byte, len(key))
			copy(cachedKey, key)
			return key, nil
		}
	}

	// Scan standard local Engine DJ application installation paths
	for _, path := range engineBinaryPaths {
		if key, err := scanBinaryForKey(path); err == nil {
			cachedKey = make([]byte, len(key))
			copy(cachedKey, key)
			return key, nil
		}
	}

	return nil, fmt.Errorf("Engine DJ Desktop installation not found: please install Engine DJ Desktop or set ENGINE_DJ_APP_PATH")
}

// scanBinaryForKey extracts the key from a supported ARM64 Mach-O initializer.
func scanBinaryForKey(binaryPath string) ([]byte, error) {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("reading binary %q: %w", binaryPath, err)
	}

	key, err := extractKey(data)
	if err != nil {
		return nil, fmt.Errorf("extracting key from %q: %w", binaryPath, err)
	}
	return key, nil
}

// Cipher provides AES-ECB encryption/decryption for stems.
type Cipher struct {
	block cipher.Block
}

// NewCipher creates a new stems cipher using dynamically derived key.
func NewCipher() (*Cipher, error) {
	key, err := DeriveKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	return &Cipher{block: block}, nil
}

// DecryptSample decrypts an AES-ECB encrypted AAC sample and removes PKCS#7 padding.
func (c *Cipher) DecryptSample(sample []byte) []byte {
	if len(sample) == 0 || len(sample)%16 != 0 {
		return sample
	}

	decrypted := make([]byte, len(sample))
	for i := 0; i < len(sample); i += 16 {
		c.block.Decrypt(decrypted[i:i+16], sample[i:i+16])
	}

	return removePKCS7Padding(decrypted)
}

// EncryptSample adds PKCS#7 padding and encrypts with AES-ECB.
func (c *Cipher) EncryptSample(sample []byte) []byte {
	padded := addPKCS7Padding(sample)
	encrypted := make([]byte, len(padded))

	for i := 0; i < len(padded); i += 16 {
		c.block.Encrypt(encrypted[i:i+16], padded[i:i+16])
	}

	return encrypted
}

func addPKCS7Padding(data []byte) []byte {
	padLen := 16 - (len(data) % 16)
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	return padded
}

func removePKCS7Padding(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > 16 || padLen > len(data) {
		return data
	}

	for i := 0; i < padLen; i++ {
		if data[len(data)-1-i] != byte(padLen) {
			return data
		}
	}

	return data[:len(data)-padLen]
}

// SetEngineAppPath sets custom Engine DJ app path for testing/verification.
func SetEngineAppPath(path string) {
	if path != "" {
		engineBinaryPaths = append([]string{
			filepath.Join(path, "Contents", "MacOS", "OfflineAnalyzer"),
			filepath.Join(path, "Contents", "MacOS", "Engine DJ"),
		}, engineBinaryPaths...)
	}
}
