// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"

	"github.com/pkg/errors"
)

var (
	errGreeting      = errors.New("zmq4: invalid greeting received")
	errSecMech       = errors.New("zmq4: invalid security mechanism")
	errBadSec        = errors.New("zmq4: invalid or unsupported security mechanism")
	errBadCmd        = errors.New("zmq4: invalid command name")
	errBadFrame      = errors.New("zmq4: invalid frame")
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

func asByte(b bool) byte {
	if b {
		return 0x01
	}
	return 0x00
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
	var data [64]byte
	_, err := io.ReadFull(r, data[:])
	if err != nil {
		return err
	}

	err = g.unmarshal(data[:])
	if err != nil {
		return err
	}

	if g.Sig.Header != sigHeader {
		return errGreeting
	}

	if g.Sig.Footer != sigFooter {
		return errGreeting
	}

	// FIXME(sbinet): handle version negotiations as per
	// https://rfc.zeromq.org/spec:23/ZMTP/#version-negotiation
	if g.Version != defaultVersion {
		return errGreeting
	}

	return nil
}

func (g *greeting) unmarshal(data []byte) error {
	if len(data) < 64 {
		return io.ErrShortBuffer
	}
	_ = data[:64]
	g.Sig.Header = data[0]
	g.Sig.Footer = data[9]
	g.Version[0] = data[10]
	g.Version[1] = data[11]
	copy(g.Mechanism[:], data[12:32])
	g.Server = data[32]
	return nil
}

func (g *greeting) write(w io.Writer) error {
	_, err := w.Write(g.marshal())
	return err
}

func (g *greeting) marshal() []byte {
	var buf [64]byte
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

const (
	sysSockType = "Socket-Type"
	sysSockID   = "Identity"
)

// MetaData describes a Conn metadata.
// The on-wire respresentation of MetaData is specified by:
//  https://rfc.zeromq.org/spec:23/ZMTP/
type MetaData struct {
	K string
	V string
}

func (md MetaData) Read(data []byte) (n int, err error) {
	klen := len(md.K)
	vlen := len(md.V)
	size := 1 + klen + 4 + vlen
	_ = data[:size] // help with bound check elision

	data[n] = byte(klen)
	n++
	n += copy(data[n:n+klen], strings.Title(md.K))
	binary.BigEndian.PutUint32(data[n:n+4], uint32(vlen))
	n += 4
	n += copy(data[n:n+vlen], md.V)
	return n, io.EOF
}

func (md *MetaData) Write(data []byte) (n int, err error) {
	klen := int(data[n])
	n++
	if klen > len(data) {
		return n, io.ErrUnexpectedEOF
	}

	md.K = strings.Title(string(data[n : n+klen]))
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

	md.V = string(data[n : n+vlen])
	n += vlen
	return n, nil
}

type flag byte

func (fl flag) hasMore() bool   { return fl&hasMoreBitFlag == hasMoreBitFlag }
func (fl flag) isLong() bool    { return fl&isLongBitFlag == isLongBitFlag }
func (fl flag) isCommand() bool { return fl&isCommandBitFlag == isCommandBitFlag }
