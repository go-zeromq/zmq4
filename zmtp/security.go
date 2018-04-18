// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmtp

import (
	"io"
)

// Security is an interface for ZMTP security mechanisms
type Security interface {
	Type() SecurityType
	// Handshake implements the ZMTP security handshake according to
	// this security mechanism.
	// see:
	//  https://rfc.zeromq.org/spec:23/ZMTP/
	//  https://rfc.zeromq.org/spec:24/ZMTP-PLAIN/
	//  https://rfc.zeromq.org/spec:25/ZMTP-CURVE/
	Handshake() error

	// Encrypt writes the encrypted form of data to w.
	Encrypt(w io.Writer, data []byte) (int, error)

	// Decrypt writes the decrypted form of data to w.
	Decrypt(w io.Writer, data []byte) (int, error)
}

// SecurityType denotes types of ZMTP security mechanisms
type SecurityType string

const (
	// NullSecurityType is an empty security mechanism
	// that does no authentication nor encryption.
	NullSecurity SecurityType = "NULL"

	// PlainSecurityType is a security mechanism that uses
	// plaintext passwords. It is a reference implementation and
	// should not be used to anything important.
	PlainSecurityType SecurityType = "PLAIN"

	// CurveSecurityType uses ZMQ_CURVE for authentication
	// and encryption.
	CurveSecurityType SecurityType = "CURVE"
)
