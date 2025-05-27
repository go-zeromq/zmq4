// Copyright 2023 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package curve

import (
	"errors"
	"fmt"
	"math/big"
)

// Z85Encoder provides the Z85 encoding for binary data.
// Z85 is a base-85 encoding designed for compactness and readability, used in ZeroMQ.
// The spec of Z85 is here: http://rfc.zeromq.org/spec:32/Z85/
//
// Z85 only encodes data of a length divisible by 4. The encoded output
// length will be 5/4 of the input length.

// Z85 encoding alphabet
var z85Encoder = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ.-:+=^!/*?&<>()[]{}@%$#")

// Lookup table for Z85 decoding
var z85Decoder [256]byte

func init() {
	// Initialize the decoder lookup table
	for i := range z85Decoder {
		z85Decoder[i] = 0xff // Invalid character marker
	}
	for i, c := range z85Encoder {
		z85Decoder[c] = byte(i)
	}
}

// Encode encodes src into Z85 encoding.
// The input length must be divisible by 4.
func Encode(src []byte) ([]byte, error) {
	// Check that we have valid input
	if len(src)%4 != 0 {
		return nil, errors.New("z85: input length must be a multiple of 4 bytes")
	}

	// Each 4 bytes of input becomes 5 bytes of output
	encodedLen := len(src) * 5 / 4
	dst := make([]byte, encodedLen)

	// Process input in 4-byte chunks
	for i, j := 0, 0; i < len(src); i += 4 {
		// Convert 4 bytes to a 32-bit integer
		value := uint32(src[i])<<24 | uint32(src[i+1])<<16 | uint32(src[i+2])<<8 | uint32(src[i+3])

		// Encode the integer as 5 characters
		for k := 4; k >= 0; k-- {
			dst[j+k] = z85Encoder[value%85]
			value /= 85
		}
		j += 5
	}

	return dst, nil
}

// Decode decodes Z85-encoded data.
// The input length must be divisible by 5.
func Decode(src []byte) ([]byte, error) {
	// Check that we have valid input
	if len(src)%5 != 0 {
		return nil, errors.New("z85: encoded length must be a multiple of 5 bytes")
	}

	// Each 5 bytes of input becomes 4 bytes of output
	decodedLen := len(src) * 4 / 5
	dst := make([]byte, decodedLen)

	// Process input in 5-byte chunks
	for i, j := 0, 0; i < len(src); i += 5 {
		// Accumulate value in base 85
		value := new(big.Int)
		base := big.NewInt(85)

		for k := 0; k < 5; k++ {
			// Check for invalid characters
			if src[i+k] >= 128 || z85Decoder[src[i+k]] == 0xff {
				return nil, fmt.Errorf("z85: invalid character '%c' at position %d", src[i+k], i+k)
			}

			digit := big.NewInt(int64(z85Decoder[src[i+k]]))
			value.Mul(value, base)
			value.Add(value, digit)
		}

		// Convert 32-bit integer to 4 bytes
		valueBytes := value.Bytes()
		// Ensure we have 4 bytes (pad with leading zeros if needed)
		padLen := 4 - len(valueBytes)
		if padLen > 0 {
			for k := 0; k < padLen; k++ {
				dst[j+k] = 0
			}
			copy(dst[j+padLen:j+4], valueBytes)
		} else {
			copy(dst[j:j+4], valueBytes[len(valueBytes)-4:])
		}

		j += 4
	}

	return dst, nil
}

// EncodeString encodes a byte slice to a Z85 string.
func EncodeString(src []byte) (string, error) {
	encoded, err := Encode(src)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// DecodeString decodes a Z85 string to a byte slice.
func DecodeString(src string) ([]byte, error) {
	return Decode([]byte(src))
}
