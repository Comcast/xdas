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

package magicbyte

//	8   |    76    |    543    |      21
//
// Reserved Encryption ContentType ContentEncoding
const (
	MagicByteLength     = 1 // bytes
	ContentEncodingBits = 2
	ContentEncodingMax  = 1<<ContentEncodingBits - 1
	ContentTypeBits     = 3
	ContentTypeMax      = 1<<ContentTypeBits - 1
	ContentTypeShift    = ContentEncodingBits
	EncryptionBits      = 2
	EncryptionMax       = 1<<EncryptionBits - 1
	EncryptionShift     = ContentEncodingBits + ContentTypeBits
	EncryptionBitmask   = EncryptionMax << EncryptionShift

// ContentBitmask      = ContentTypeMax<<ContentTypeBits | ContentEncodingMax
)

const (
	ContentEncodingNone = iota
	ContentEncodingZstd
	ContentEncodingZlib
)

var contentEncodingText = map[int]string{
	ContentEncodingZstd: "zstd",
	ContentEncodingZlib: "zlib",
}

var contentEncodingCode = map[string]int{
	"":     ContentEncodingNone,
	"zstd": ContentEncodingZstd,
	"zlib": ContentEncodingZlib,
}

const (
	ContentTypeUnknown = iota
	ContentTypeJson
	ContentTypeProtoBuf
	//	ContentTypeString
)

var contentTypeText = map[int]string{
	ContentTypeUnknown:  "application/octet-stream",
	ContentTypeJson:     "application/json",
	ContentTypeProtoBuf: "application/x-protobuf",
	//	ContentTypeString:   "application/json",
}

var contentTypeCode = map[string]int{
	"":                                ContentTypeUnknown,
	"application/json":                ContentTypeJson,
	"json":                            ContentTypeJson,
	"application/x-protobuf":          ContentTypeProtoBuf,
	"application/vnd.google.protobuf": ContentTypeProtoBuf,
	"protobuf":                        ContentTypeProtoBuf,
	"application/octet-stream":        ContentTypeUnknown,
}

// MagicByte stores the content encoding/type and encryption method
// type MagicByte byte
type MagicByte struct {
	cev        int
	ctv        int
	encryption int
}

// New returns MagicByte based on content/encoding type and encryption value
func New(contentEncoding, contentType string, encryption int) MagicByte {
	cev := contentEncodingCode[contentEncoding]
	ctv := contentTypeCode[contentType]
	return NewMagicByte(cev, ctv, encryption)
}

// NewFrom returns a new MagicByte from an existing magicByte
func NewFrom(magicByte byte) MagicByte {
	return MagicByte{
		cev:        int(magicByte & ContentEncodingMax),
		ctv:        int(magicByte >> ContentTypeShift & ContentTypeMax),
		encryption: int(magicByte >> EncryptionShift & EncryptionMax),
	}
}

// NewMagicByte returns MagicByte from cev, ctv and encryption values
func NewMagicByte(cev, ctv, encryption int) MagicByte {
	return MagicByte{
		cev:        cev,
		ctv:        ctv,
		encryption: encryption,
	}
}

// GetContentTypeText returns the text string of Content-type for ctv
func GetContentTypeText(ctv int) string {
	return contentTypeText[ctv]
}

// GetContentEncodingText returns the text string of Content-encoding for cev
func GetContentEncodingText(cev int) string {
	return contentEncodingText[cev]
}

// Get returns the value of magicByte
func (m *MagicByte) Get() byte {
	return byte(m.encryption<<EncryptionShift | m.ctv<<ContentTypeShift | m.cev)
}

// GetCEV returns an integer value of ContentConding
func (m *MagicByte) GetCEV() int {
	return m.cev
}

// GetCTV returns an integer value of ContentType
func (m *MagicByte) GetCTV() int {
	return m.ctv
}

// GetContentEncoding returns HTTP Content-encoding
func (m *MagicByte) GetContentEncoding() string {
	return contentEncodingText[m.GetCEV()]
}

// GetContentType returns HTTP Content-type
func (m *MagicByte) GetContentType() string {
	return contentTypeText[m.GetCTV()]
}

// GetEncryption returns an integer value of encryption used
func (m *MagicByte) GetEncryption() int {
	return m.encryption
}

// AddEncrypt adds encryption to MagicByte
func (m *MagicByte) AddEncrypt(encryption int) {
	// if encryption > EncryptionMax {
	// 	encryption = 0
	// }
	m.encryption = encryption
}

// Header is an interface that both http.Header and textproto.MIMEHeader implements
type Header interface {
	Set(key, value string)
}

// SetContentHeaders sets the appropriate Content-type and Content-encoding
func (m *MagicByte) SetContentHeaders(h Header) {
	h.Set("Content-type", m.GetContentType())
	if contentEncoding := m.GetContentEncoding(); contentEncoding != "" {
		h.Set("Content-encoding", contentEncoding)
	}
}
