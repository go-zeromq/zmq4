// Copyright 2023 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package curve

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestZ85EncodeDecode(t *testing.T) {

	decodeString, _ := hex.DecodeString("9493c171319a11c5469db7e81bae204768efa826b9a2f144ff4a581cdb3eed4b")
	testCases := []struct {
		name    string
		input   []byte
		encoded string
	}{
		{
			name:    "HelloWorld",
			input:   []byte{0x86, 0x4F, 0xD2, 0x6F, 0xB5, 0x59, 0xF7, 0x5B},
			encoded: "HelloWorld",
		},
		{
			name:    "EmptyString",
			input:   []byte{0, 0, 0, 0},
			encoded: "00000",
		},
		{
			name:    "Binary",
			input:   decodeString,
			encoded: "L-@}6f}5Y[mXd3L8)gqNxZ.(DXUwQZ%4lxk*DL0$",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test encoding
			enc, err := Encode(tc.input)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			if string(enc) != tc.encoded {
				t.Errorf("Encode result mismatch: got %q, want %q", string(enc), tc.encoded)
			}

			// Test decoding
			dec, err := Decode([]byte(tc.encoded))
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if !bytes.Equal(dec, tc.input) {
				t.Errorf("Decode result mismatch: got %v, want %v", dec, tc.input)
			}

			// Test EncodeString and DecodeString convenience functions
			encStr, err := EncodeString(tc.input)
			if err != nil {
				t.Fatalf("EncodeString failed: %v", err)
			}

			if encStr != tc.encoded {
				t.Errorf("EncodeString result mismatch: got %q, want %q", encStr, tc.encoded)
			}

			decBytes, err := DecodeString(tc.encoded)
			if err != nil {
				t.Fatalf("DecodeString failed: %v", err)
			}

			if !bytes.Equal(decBytes, tc.input) {
				t.Errorf("DecodeString result mismatch: got %v, want %v", decBytes, tc.input)
			}
		})
	}
}

func TestZ85InvalidInput(t *testing.T) {
	// Test invalid input lengths for encoding
	_, err := Encode([]byte{1, 2, 3}) // Not divisible by 4
	if err == nil {
		t.Error("Expected error for input length not divisible by 4, got nil")
	}

	// Test invalid input lengths for decoding
	_, err = Decode([]byte{1, 2, 3, 4}) // Not divisible by 5
	if err == nil {
		t.Error("Expected error for input length not divisible by 5, got nil")
	}

	// Test invalid characters for decoding
	_, err = Decode([]byte("Hello~World")) // ~ is not in the Z85 alphabet
	if err == nil {
		t.Error("Expected error for invalid character, got nil")
	}
}

// Test the examples from the Z85 spec: http://rfc.zeromq.org/spec:32/Z85/
func TestZ85SpecExamples(t *testing.T) {
	// Example 1: a 32-byte CURVE key encoded with Z85
	key := []byte{
		0x8E, 0x0B, 0xDD, 0x69, 0x76, 0x28, 0xB9, 0x1D,
		0x8F, 0x24, 0x55, 0x87, 0xEE, 0x95, 0xC5, 0xB0,
		0x4D, 0x48, 0x96, 0x3F, 0x79, 0x25, 0x98, 0x77,
		0xB4, 0x9C, 0xD9, 0x06, 0x3A, 0xEA, 0xD3, 0xB7,
	}

	expected := "JTKVSB%%)wK0E.X)V>+}o?pNmC{O&4W4b!Ni{Lh6"

	encoded, err := EncodeString(key)
	if err != nil {
		t.Fatalf("EncodeString failed: %v", err)
	}

	if encoded != expected {
		t.Errorf("Z85 spec example encoding mismatch: got %q, want %q", encoded, expected)
	}

	decoded, err := DecodeString(expected)
	if err != nil {
		t.Fatalf("DecodeString failed: %v", err)
	}

	if !bytes.Equal(decoded, key) {
		t.Errorf("Z85 spec example decoding mismatch: got %v, want %v", decoded, key)
	}
}

func BenchmarkZ85Encode(b *testing.B) {
	data := bytes.Repeat([]byte{1, 2, 3, 4}, 256) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Encode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkZ85Decode(b *testing.B) {
	data := bytes.Repeat([]byte{1, 2, 3, 4}, 256) // 1KB
	encoded, _ := Encode(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decode(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}
