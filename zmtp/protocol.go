// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmtp

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"

	"github.com/pkg/errors"
)

var (
	errGreeting      = errors.New("zmtp: invalid greeting received")
	errSecMech       = errors.New("zmtp: invalid security mechanism")
	errBadSec        = errors.New("zmtp: invalid or unsupported security mechanism")
	errBadCmd        = errors.New("zmtp: invalid command name")
	errBadFrame      = errors.New("zmtp: invalid frame")
	errOverflow      = errors.New("zmtp: overflow")
	errEmptyAppMDKey = errors.New("zmtp: empty application metadata key")
	errDupAppMDKey   = errors.New("zmtp: duplicate application metadata key")
	errBoolCnv       = errors.New("zmtp: invalid byte to bool conversion")
	errMoreCmd       = errors.New("zmtp: MORE not supported") // FIXME(sbinet)
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

// command is a ZMTP command as per:
//  https://rfc.zeromq.org/spec:23/ZMTP/#formal-grammar
type command struct {
	Name string
	Body []byte
}

func (cmd *command) unmarshalZMTP(data []byte) error {
	if len(data) == 0 {
		return io.ErrUnexpectedEOF
	}
	n := int(data[0])
	if n > len(data)-1 {
		return errBadCmd
	}
	cmd.Name = string(data[1 : n+1])
	cmd.Body = data[n+1:]
	return nil
}

func (cmd *command) marshalZMTP() ([]byte, error) {
	n := len(cmd.Name)
	if n > 255 {
		return nil, errBadCmd
	}

	buf := make([]byte, 0, 1+n+len(cmd.Body))
	buf = append(buf, byte(n))
	buf = append(buf, []byte(cmd.Name)...)
	buf = append(buf, cmd.Body...)
	return buf, nil
}

// ZMTP commands as per:
//  https://rfc.zeromq.org/spec:23/ZMTP/#commands
const (
	cmdCancel    = "CANCEL"
	cmdError     = "ERROR"
	cmdHello     = "HELLO"
	cmdPing      = "PING"
	cmdPong      = "PONG"
	cmdReady     = "READY"
	cmdSubscribe = "SUBSCRIBE"
)

const (
	sysSockType = "Socket-Type"
	sysSockID   = "Identity"
)

type metaData struct {
	k string
	v string
}

func (md metaData) Read(data []byte) (n int, err error) {
	klen := len(md.k)
	vlen := len(md.v)
	size := 1 + klen + 4 + vlen
	_ = data[:size] // help with bound check elision

	data[n] = byte(klen)
	n++
	n += copy(data[n:n+klen], strings.Title(md.k))
	binary.BigEndian.PutUint32(data[n:n+4], uint32(vlen))
	n += 4
	n += copy(data[n:n+vlen], md.v)
	return n, io.EOF
}

func (md *metaData) Write(data []byte) (n int, err error) {
	klen := int(data[n])
	n++
	if klen > len(data) {
		return n, io.ErrUnexpectedEOF
	}

	md.k = strings.Title(string(data[n : n+klen]))
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

	md.v = string(data[n : n+vlen])
	n += vlen
	return n, nil
}

type flag byte

func (f flag) hasMore() bool    { return f == 0x1 || f == 0x3 }
func (fl flag) isLong() bool    { return fl == 0x2 || fl == 0x3 || fl == 0x6 }
func (fl flag) isCommand() bool { return fl == 0x4 || fl == 0x6 }
