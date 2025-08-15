// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package rediscrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

// RedisCrypto is used to encrypt/decrypt data to/from Redis
type RedisCrypto interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AesGCM implements RedisCrypto
type AesGCM struct {
	gcm      cipher.AEAD
	key      []byte
	overhead int
}

var globalDefaultCrypto RedisCrypto

// Encrypt uses globalDefaultCrypto to encrypt. Must call Init first.
var Encrypt func(plaintext []byte) ([]byte, error)

// Decrypt uses globalDefaultCrypto to decrypt. Must call Init first.
var Decrypt func(ciphertext []byte) ([]byte, error)

// Init must be called before using global Encrypt and Decrypt
func Init(cryptoType string, hexKey []string) (RedisCrypto, error) {
	switch strings.ToUpper(cryptoType) {
	case "AESGCM":
		crypto, err := NewAesGCM(hexKey)
		if err != nil {
			return nil, err
		}
		globalDefaultCrypto = crypto
		Encrypt = globalDefaultCrypto.Encrypt
		Decrypt = globalDefaultCrypto.Decrypt
		return globalDefaultCrypto, err
	default:
		panic("Unkown cryptoType: " + cryptoType)
		// return nil, errors.New("Unkown cryptoType: " + cryptoType)
	}
}

// NewAesGCM returns AESGCM that implements RdisCrypto. Only 1st hexkey entry is used for now
func NewAesGCM(hexKey []string) (RedisCrypto, error) {
	aesGCM := AesGCM{}
	for index, value := range hexKey {
		if len(value) != 64 { // need 64 bytes hex for AES-256
			return nil, errors.New("Invalid EncryptionKey" + value)
		}
		if index == 0 {
			key, err := hex.DecodeString(value)
			if err != nil {
				return nil, err
			}
			block, err := aes.NewCipher(key)
			if err != nil {
				return nil, err
			}
			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return nil, err
			}

			aesGCM.key = key
			//	aesGCM.block = block
			aesGCM.gcm = gcm
			aesGCM.overhead = gcm.Overhead() + gcm.NonceSize()
		}
	}

	return &aesGCM, nil
}

// Encrypt accepts plaintext and returns AES256 AEAD (GCM) encrypted ciphertext
// there is a total 28 bytes overhead, of which 16 is GCM overhead, and 12 is embedded nonce
// output: <nonce><GCM overhead><ciphertext>
func (a *AesGCM) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, a.gcm.NonceSize(), len(plaintext)+a.overhead)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := a.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt accepts AES256 encrypted ciphertext and returns plaintext
func (a *AesGCM) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < a.overhead {
		return nil, errors.New("not valid AES GCM encrypted data")
	}
	nonce := ciphertext[:a.gcm.NonceSize()]
	ciphertext = ciphertext[a.gcm.NonceSize():]
	plaintext, err := a.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
