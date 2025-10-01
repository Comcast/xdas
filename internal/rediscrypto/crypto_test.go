/*
 * Copyright 2025 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rediscrypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"testing"
)

// var aesCBC RedisCrypto
var aesGCM RedisCrypto
var Small, Medium, Large, XLarge []byte

// var SmallCBCCipher, MediumCBCCipher, LargeCBCCipher, XLargeCBCCipher []byte
var SmallGCMCipher, MediumGCMCipher, LargeGCMCipher, XLargeGCMCipher []byte

func init() {
	key := make([]byte, 32) // 32 bytes for AES-256
	io.ReadFull(rand.Reader, key)
	hexKeys := []string{hex.EncodeToString(key)}
	// aesCBC, _ = NewAesCBC(hexKeys)
	Init("AesGCM", hexKeys)
	aesGCM, _ = NewAesGCM(hexKeys)
	Small = make([]byte, 10)
	Medium = make([]byte, 100)
	Large = make([]byte, 600)
	XLarge = make([]byte, 6000)
	io.ReadFull(rand.Reader, Small)
	io.ReadFull(rand.Reader, Medium)
	io.ReadFull(rand.Reader, Large)
	io.ReadFull(rand.Reader, XLarge)
	SmallGCMCipher, _ = aesGCM.Encrypt(Small)
	MediumGCMCipher, _ = aesGCM.Encrypt(Medium)
	LargeGCMCipher, _ = aesGCM.Encrypt(Large)
	XLargeGCMCipher, _ = aesGCM.Encrypt(XLarge)
}

func TestAesGCM(t *testing.T) {
	tests := [][]byte{
		{},
		{0},
		{0, 1, 2, 3, 4, 5, 6},
		[]byte("this is a test"),
		[]byte("x100-12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345"),
	}
	for _, tt := range tests {
		tt := tt
		// original := []byte(value)
		ciphertext, err := aesGCM.Encrypt(tt)
		if err != nil {
			t.Error(err)
		}
		plaintext, err := aesGCM.Decrypt(ciphertext)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(tt, plaintext) {
			if len(tt) == 0 && len(plaintext) == 0 {
				continue
			}
			t.Error("Got", plaintext, "want", tt)
		}
	}
}

func TestGlobalAesGCM(t *testing.T) {
	tests := [][]byte{
		{},
		{0},
		{0, 1, 2, 3, 4, 5, 6},
		[]byte("this is a test"),
		[]byte("x100-12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345"),
	}
	for _, tt := range tests {
		// original := []byte(value)
		ciphertext, err := Encrypt(tt)
		if err != nil {
			t.Error(err)
		}
		plaintext, err := Decrypt(ciphertext)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(tt, plaintext) {
			if len(tt) == 0 && len(plaintext) == 0 {
				continue
			}
			t.Error("Got", plaintext, "want", tt)
		}
	}
}

func benchmarkAesGCMEncrypt(plaintext []byte, b *testing.B) {
	data := make([]byte, len(plaintext))
	for n := 0; n < b.N; n++ {
		copy(data, plaintext)
		aesGCM.Encrypt(data)
	}
}

func benchmarkAesGCMDecrypt(ciphertext []byte, b *testing.B) {
	data := make([]byte, len(ciphertext))
	for n := 0; n < b.N; n++ {
		copy(data, ciphertext)
		aesGCM.Decrypt(data)
	}
}

func BenchmarkAesGCMSmallEnc(b *testing.B)  { benchmarkAesGCMEncrypt(Small, b) }
func BenchmarkAesGCMMediumEnc(b *testing.B) { benchmarkAesGCMEncrypt(Medium, b) }
func BenchmarkAesGCMLargeEnc(b *testing.B)  { benchmarkAesGCMEncrypt(Large, b) }
func BenchmarkAesGCMXLargeEnc(b *testing.B) { benchmarkAesGCMEncrypt(XLarge, b) }
func BenchmarkAesGCMSmallDec(b *testing.B)  { benchmarkAesGCMDecrypt(SmallGCMCipher, b) }
func BenchmarkAesGCMMediumDec(b *testing.B) { benchmarkAesGCMDecrypt(MediumGCMCipher, b) }
func BenchmarkAesGCMLargeDec(b *testing.B)  { benchmarkAesGCMDecrypt(LargeGCMCipher, b) }
func BenchmarkAesGCMXLargeDec(b *testing.B) { benchmarkAesGCMDecrypt(XLargeGCMCipher, b) }
