// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plain provides the ZeroMQ PLAIN security mechanism as specified by:
// https://rfc.zeromq.org/spec:24/ZMTP-PLAIN/
package plain

import (
	"io"

	"github.com/go-zeromq/zmq4"
	"github.com/pkg/errors"
)

// security implements the PLAIN security mechanism.
type security struct {
	user []byte
	pass []byte
}

// Security returns a value that implements the PLAIN security mechanism
func Security(user, pass string) zmq4.Security {
	return &security{[]byte(user), []byte(pass)}
}

// Type returns the security mechanism type.
func (security) Type() zmq4.SecurityType {
	return zmq4.PlainSecurity
}

// Handshake implements the ZMTP security handshake according to
// this security mechanism.
// see:
//  https://rfc.zeromq.org/spec:23/ZMTP/
//  https://rfc.zeromq.org/spec:24/ZMTP-PLAIN/
//  https://rfc.zeromq.org/spec:25/ZMTP-CURVE/
func (sec *security) Handshake(conn *zmq4.Conn, server bool) error {
	switch {
	case server:
		cmd, err := conn.RecvCmd()
		if err != nil {
			return errors.WithMessage(err, "could not receive HELLO from client")
		}

		if cmd.Name != zmq4.CmdHello {
			return errors.Errorf("security/plain: expected HELLO command")
		}

		// FIXME(sbinet): perform a real authentication
		err = validateHello(cmd.Body)
		if err != nil {
			conn.SendCmd(zmq4.CmdError, []byte("invalid")) // FIXME(sbinet) correct ERROR reason
			return errors.WithMessage(err, "could not authenticate client")
		}

		err = conn.SendCmd(zmq4.CmdWelcome, nil)
		if err != nil {
			return errors.WithMessage(err, "could not send WELCOME to client")
		}

		cmd, err = conn.RecvCmd()
		if err != nil {
			return errors.WithMessage(err, "could not receive INITIATE from client")
		}

		err = conn.Peer.Meta.UnmarshalZMTP(cmd.Body)
		if err != nil {
			return errors.WithMessage(err, "could not unmarshal peer metadata")
		}

		raw, err := conn.Meta.MarshalZMTP()
		if err != nil {
			conn.SendCmd(zmq4.CmdError, []byte("invalid")) // FIXME(sbinet) correct ERROR reason
			return errors.WithMessage(err, "could not serialize metadata")
		}

		err = conn.SendCmd(zmq4.CmdReady, raw)
		if err != nil {
			return errors.WithMessage(err, "could not send READY to client")
		}

	case !server:
		hello := make([]byte, 0, len(sec.user)+len(sec.pass)+2)
		hello = append(hello, byte(len(sec.user)))
		hello = append(hello, sec.user...)
		hello = append(hello, byte(len(sec.pass)))
		hello = append(hello, sec.pass...)

		err := conn.SendCmd(zmq4.CmdHello, hello)
		if err != nil {
			return errors.WithMessage(err, "could not send HELLO to server")
		}

		cmd, err := conn.RecvCmd()
		if err != nil {
			return errors.WithMessage(err, "could not receive WELCOME from server")
		}
		if cmd.Name != zmq4.CmdWelcome {
			conn.SendCmd(zmq4.CmdError, []byte("invalid command")) // FIXME(sbinet) correct ERROR reason
			return errors.WithMessage(err, "expected a WELCOME command from server")
		}

		raw, err := conn.Meta.MarshalZMTP()
		if err != nil {
			conn.SendCmd(zmq4.CmdError, []byte("internal error")) // FIXME(sbinet) correct ERROR reason
			return errors.WithMessage(err, "could not serialize metadata")
		}

		err = conn.SendCmd(zmq4.CmdInitiate, raw)
		if err != nil {
			return errors.WithMessage(err, "could not send INITIATE to server")
		}

		cmd, err = conn.RecvCmd()
		if err != nil {
			return errors.WithMessage(err, "could not receive READY from server")
		}
		if cmd.Name != zmq4.CmdReady {
			conn.SendCmd(zmq4.CmdError, []byte("invalid command")) // FIXME(sbinet) correct ERROR reason
			return errors.WithMessage(err, "expected a READY command from server")
		}

		err = conn.Peer.Meta.UnmarshalZMTP(cmd.Body)
		if err != nil {
			return errors.WithMessage(err, "could not unmarshal peer metadata")
		}

		sec.user = nil
		sec.pass = nil
	}
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

// validateHello validates the user/passwd credentials.
func validateHello(body []byte) error {
	//	n := int(body[0])
	//	user := body[1 : 1+n]
	//	body = body[1+n:]
	//	n = int(body[0])
	//	pass := body[1 : 1+n]
	//	body = body[1+n:]
	//	log.Printf("user=%q, pass=%q, body=%q", user, pass, body)
	return nil
}

var (
	_ zmq4.Security = (*security)(nil)
)
