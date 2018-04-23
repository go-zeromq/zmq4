// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package null provides the ZeroMQ NULL security mechanism
package null

import (
	"io"

	"github.com/go-zeromq/zmq4"
)

// security implements the NULL security mechanism.
type security struct{}

// Security returns a value that implements the NULL security mechanism
func Security() zmq4.Security {
	return security{}
}

// Type returns the security mechanism type.
func (security) Type() zmq4.SecurityType {
	return zmq4.NullSecurity
}

// Handshake implements the ZMTP security handshake according to
// this security mechanism.
// see:
//  https://rfc.zeromq.org/spec:23/ZMTP/
//  https://rfc.zeromq.org/spec:24/ZMTP-PLAIN/
//  https://rfc.zeromq.org/spec:25/ZMTP-CURVE/
func (security) Handshake() error {
	return nil
}

// Encrypt writes the encrypted form of data to w.
func (security) Encrypt(w io.Writer, data []byte) (int, error) {
	return w.Write(data)
}

// Decrypt writes the decrypted form of data to w.
func (security) Decrypt(w io.Writer, data []byte) (int, error) {
	return w.Write(data)
}
