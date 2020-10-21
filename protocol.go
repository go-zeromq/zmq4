// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	errGreeting      = errors.New("zmq4: invalid greeting received")
	errSecMech       = errors.New("zmq4: invalid security mechanism")
	errBadSec        = errors.New("zmq4: invalid or unsupported security mechanism")
	ErrBadCmd        = errors.New("zmq4: invalid command name")
	ErrBadFrame      = errors.New("zmq4: invalid frame")
	errOverflow      = errors.New("zmq4: overflow")
	errEmptyAppMDKey = errors.New("zmq4: empty application metadata key")
	errDupAppMDKey   = errors.New("zmq4: duplicate application metadata key")
	errBoolCnv       = errors.New("zmq4: invalid byte to bool conversion")
)

const (
	sigHeader = 0xFF
	sigFooter = 0x7F

	majorVersion uint8 = 3
	minorVersion uint8 = 0

	hasMoreBitFlag   = 0x1
	isLongBitFlag    = 0x2
	isCommandBitFlag = 0x4

	zmtpMsgLen = 64
)

var (
	defaultVersion = [2]uint8{
		majorVersion,
		minorVersion,
	}
)

const (
	maxUint   = ^uint(0)
	maxInt    = int(maxUint >> 1)
	maxUint64 = ^uint64(0)
	maxInt64  = int64(maxUint64 >> 1)
)

func asString(slice []byte) string {
	i := bytes.IndexByte(slice, 0)
	if i < 0 {
		i = len(slice)
	}
	return string(slice[:i])
}

func asBool(b byte) (bool, error) {
	switch b {
	case 0x00:
		return false, nil
	case 0x01:
		return true, nil
	}

	return false, errBoolCnv
}

type greeting struct {
	Sig struct {
		Header byte
		_      [8]byte
		Footer byte
	}
	Version   [2]uint8
	Mechanism [20]byte
	Server    byte
	_         [31]byte
}

func (g *greeting) read(r io.Reader) error {
	var data [zmtpMsgLen]byte
	_, err := io.ReadFull(r, data[:])
	if err != nil {
		return fmt.Errorf("could not read ZMTP greeting: %w", err)
	}

	g.unmarshal(data[:])

	if g.Sig.Header != sigHeader {
		return fmt.Errorf("invalid ZMTP signature header: %w", errGreeting)
	}

	if g.Sig.Footer != sigFooter {
		return fmt.Errorf("invalid ZMTP signature footer: %w", errGreeting)
	}

	if !g.validate(defaultVersion) {
		return fmt.Errorf(
			"invalid ZMTP version (got=%v, want=%v): %w",
			g.Version, defaultVersion, errGreeting,
		)
	}

	return nil
}

func (g *greeting) unmarshal(data []byte) {
	_ = data[:zmtpMsgLen]
	g.Sig.Header = data[0]
	g.Sig.Footer = data[9]
	g.Version[0] = data[10]
	g.Version[1] = data[11]
	copy(g.Mechanism[:], data[12:32])
	g.Server = data[32]
}

func (g *greeting) write(w io.Writer) error {
	_, err := w.Write(g.marshal())
	return err
}

func (g *greeting) marshal() []byte {
	var buf [zmtpMsgLen]byte
	buf[0] = g.Sig.Header
	// padding 1 ignored
	buf[9] = g.Sig.Footer
	buf[10] = g.Version[0]
	buf[11] = g.Version[1]
	copy(buf[12:32], g.Mechanism[:])
	buf[32] = g.Server
	// padding 2 ignored
	return buf[:]
}

func (g *greeting) validate(ref [2]uint8) bool {
	switch {
	case g.Version == ref:
		return true
	case g.Version[0] > ref[0] ||
		g.Version[0] == ref[0] && g.Version[1] > ref[1]:
		// accept higher protocol values
		return true
	case g.Version[0] < ref[0] ||
		g.Version[0] == ref[0] && g.Version[1] < ref[1]:
		// FIXME(sbinet): handle version negotiations as per
		// https://rfc.zeromq.org/spec:23/ZMTP/#version-negotiation
		return false
	default:
		return false
	}
}

const (
	sysSockType = "Socket-Type"
	sysSockID   = "Identity"
)

// Metadata is describing a Conn's metadata information.
type Metadata map[string]string

// MarshalZMTP marshals MetaData to ZMTP encoded data.
func (md Metadata) MarshalZMTP() ([]byte, error) {
	buf := new(bytes.Buffer)
	keys := make(map[string]struct{})

	for k, v := range md {
		if len(k) == 0 {
			return nil, errEmptyAppMDKey
		}

		key := strings.ToLower(k)
		if _, dup := keys[key]; dup {
			return nil, errDupAppMDKey
		}

		keys[key] = struct{}{}
		switch k {
		case sysSockID, sysSockType:
			if _, err := io.Copy(buf, Property{K: k, V: v}); err != nil {
				return nil, err
			}
		default:
			if _, err := io.Copy(buf, Property{K: "X-" + key, V: v}); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalZMTP unmarshals MetaData from a ZMTP encoded data.
func (md *Metadata) UnmarshalZMTP(p []byte) error {
	i := 0
	for i < len(p) {
		var kv Property
		n, err := kv.Write(p[i:])
		if err != nil {
			return err
		}
		i += n

		name := strings.Title(kv.K)
		(*md)[name] = kv.V
	}
	return nil
}

// Property describes a Conn metadata's entry.
// The on-wire respresentation of Property is specified by:
//  https://rfc.zeromq.org/spec:23/ZMTP/
type Property struct {
	K string
	V string
}

func (prop Property) Read(data []byte) (n int, err error) {
	klen := len(prop.K)
	vlen := len(prop.V)
	size := 1 + klen + 4 + vlen
	_ = data[:size] // help with bound check elision

	data[n] = byte(klen)
	n++
	n += copy(data[n:n+klen], strings.Title(prop.K))
	binary.BigEndian.PutUint32(data[n:n+4], uint32(vlen))
	n += 4
	n += copy(data[n:n+vlen], prop.V)
	return n, io.EOF
}

func (prop *Property) Write(data []byte) (n int, err error) {
	klen := int(data[n])
	n++
	if klen > len(data) {
		return n, io.ErrUnexpectedEOF
	}

	prop.K = strings.Title(string(data[n : n+klen]))
	n += klen

	v := binary.BigEndian.Uint32(data[n : n+4])
	n += 4
	if uint64(v) > uint64(maxInt) {
		return n, errOverflow
	}

	vlen := int(v)
	if n+vlen > len(data) {
		return n, io.ErrUnexpectedEOF
	}

	prop.V = string(data[n : n+vlen])
	n += vlen
	return n, nil
}

type flag byte

func (fl flag) hasMore() bool   { return fl&hasMoreBitFlag == hasMoreBitFlag }
func (fl flag) isLong() bool    { return fl&isLongBitFlag == isLongBitFlag }
func (fl flag) isCommand() bool { return fl&isCommandBitFlag == isCommandBitFlag }
