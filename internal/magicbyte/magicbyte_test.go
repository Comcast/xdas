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

import (
	"testing"
)

func TestMagicCreate(t *testing.T) {
	tests := []struct {
		contentEncoding string
		contentType     string
		encryption      int
		result          byte
	}{
		{"", "dummy", 0, 0},
		{"zstd", "", 1, 33},
		{"zstd", "dummy", 2, 65},
		{"zlib", "", 3, 98},
		{"", "application/json", 0, 4},
		{"dummy", "application/json", 1, 36},
		{"zstd", "application/json", 2, 69},
		{"zlib", "application/json", 3, 102},
		{"", "application/x-protobuf", 1, 40},
		{"zstd", "application/x-protobuf", 1, 41},
	}

	for _, tt := range tests {
		tt := tt
		m := New(tt.contentEncoding, tt.contentType, tt.encryption)
		r := NewFrom(tt.result)

		if m.Get() != tt.result {
			t.Errorf("MagicCreate (%v)+(%v)+(%v) was incorrect, got: %d, %b, want: %d.",
				tt.contentEncoding, tt.contentType, tt.encryption, m, m.Get(), tt.result)
		}
		if m.GetCEV() != r.GetCEV() || m.GetCTV() != r.GetCTV() || m.GetEncryption() != r.GetEncryption() {
			t.Errorf("MagicCreate not matching (%v)+(%v)+(%v) was incorrect, got: %d, %b, want: %d.",
				tt.contentEncoding, tt.contentType, tt.encryption, m, m.Get(), tt.result)
		}
	}
}

func TestMagicGet(t *testing.T) {
	tests := []struct {
		contentEncoding       string
		contentType           string
		encryption            int
		resultContentEncoding string
		resultContentType     string
	}{
		{"", "dummy", 0, "", "application/octet-stream"},
		{"", "application/octet-stream", 0, "", "application/octet-stream"},
		{"zstd", "", 1, "zstd", "application/octet-stream"},
		{"zstd", "dummy", 2, "zstd", "application/octet-stream"},
		{"zlib", "", 3, "zlib", "application/octet-stream"},
		{"", "application/json", 0, "", "application/json"},
		{"dummy", "application/json", 1, "", "application/json"},
		{"zstd", "application/json", 2, "zstd", "application/json"},
		{"zstd", "json", 2, "zstd", "application/json"},
		{"zlib", "application/json", 3, "zlib", "application/json"},
		{"", "application/x-protobuf", 0, "", "application/x-protobuf"},
		{"zstd", "application/vnd.google.protobuf", 0, "zstd", "application/x-protobuf"},
		{"zst", "protobuf", 0, "", "application/x-protobuf"},
	}

	for _, tt := range tests {
		tt := tt
		m := New(tt.contentEncoding, tt.contentType, tt.encryption)

		if m.GetContentEncoding() != tt.resultContentEncoding {
			t.Errorf("MagicGetContentEncoding (%v)+(%v) was incorrect, got: %v, want: %v.",
				tt.contentEncoding, tt.contentType, m.GetContentEncoding(), tt.resultContentEncoding)
		}
		if m.GetContentType() != tt.resultContentType {
			t.Errorf("MagicGetContentType (%v)+(%v) was incorrect, got: %v, want: %v.",
				tt.contentEncoding, tt.contentType, m.GetContentType(), tt.resultContentType)
		}
		if m.GetEncryption() != tt.encryption {
			t.Errorf("MagicEncryptionIndex was incorrect, got: %v, want: %v.",
				m.GetEncryption(), tt.encryption)
		}
	}
}
