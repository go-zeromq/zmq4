// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"io"
)

// Security is an interface for ZMTP security mechanisms
type Security interface {
	// Type returns the security mechanism type.
	Type() SecurityType

	// Handshake implements the ZMTP security handshake according to
	// this security mechanism.
	// see:
	//  https://rfc.zeromq.org/spec:23/ZMTP/
	//  https://rfc.zeromq.org/spec:24/ZMTP-PLAIN/
	//  https://rfc.zeromq.org/spec:25/ZMTP-CURVE/
	Handshake(conn *Conn, server bool) error

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

	// PlainSecurity is a security mechanism that uses
	// plaintext passwords. It is a reference implementation and
	// should not be used to anything important.
	PlainSecurity SecurityType = "PLAIN"

	// CurveSecurity uses ZMQ_CURVE for authentication
	// and encryption.
	CurveSecurity SecurityType = "CURVE"
)

// security implements the NULL security mechanism.
type nullSecurity struct{}

// Type returns the security mechanism type.
func (nullSecurity) Type() SecurityType {
	return NullSecurity
}

// Handshake implements the ZMTP security handshake according to
// this security mechanism.
// see:
//  https://rfc.zeromq.org/spec:23/ZMTP/
//  https://rfc.zeromq.org/spec:24/ZMTP-PLAIN/
//  https://rfc.zeromq.org/spec:25/ZMTP-CURVE/
func (nullSecurity) Handshake(conn *Conn, server bool) error {
	err := conn.sendMD(conn.Meta)
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not send metadata to peer")
	}

	conn.Peer.Meta, err = conn.recvMD()
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not recv metadata from peer")
	}

	return nil
}

// Encrypt writes the encrypted form of data to w.
func (nullSecurity) Encrypt(w io.Writer, data []byte) (int, error) {
	return w.Write(data)
}

// Decrypt writes the decrypted form of data to w.
func (nullSecurity) Decrypt(w io.Writer, data []byte) (int, error) {
	return w.Write(data)
}

var (
	_ Security = (*nullSecurity)(nil)
)
