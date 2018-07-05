// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// Conn implements the ZeroMQ Message Transport Protocol as defined
// in https://rfc.zeromq.org/spec:23/ZMTP/.
type Conn struct {
	typ    SocketType
	id     SocketIdentity
	rw     io.ReadWriteCloser
	sec    Security
	server bool
	meta   map[string]string
	peer   struct {
		server bool
		meta   map[string]string
	}

	mu     sync.RWMutex
	topics map[string]struct{} // set of subscribed topics
}

func (c *Conn) Close() error {
	return c.rw.Close()
}

func (c *Conn) Read(p []byte) (int, error) {
	return c.rw.Read(p)
}

func (c *Conn) Write(p []byte) (int, error) {
	return c.rw.Write(p)
}

// Open opens a ZMTP connection over rw with the given security, socket type and identity.
// Open performs a complete ZMTP handshake.
func Open(rw io.ReadWriteCloser, sec Security, sockType SocketType, sockID SocketIdentity, server bool) (*Conn, error) {
	if rw == nil {
		return nil, errors.Errorf("zmq4: invalid nil read-writer")
	}

	if sec == nil {
		return nil, errors.Errorf("zmq4: invalid nil security")
	}

	conn := &Conn{
		typ:    sockType,
		id:     sockID,
		rw:     rw,
		sec:    sec,
		server: server,
		meta:   nil,
		topics: make(map[string]struct{}),
	}

	err := conn.init(sec)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// init performs a ZMTP handshake over an io.ReadWriter
func (conn *Conn) init(sec Security) error {
	var err error

	err = conn.greet(conn.server)
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not exchange greetings")
	}

	err = conn.sec.Handshake(conn, conn.server)
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not perform security handshake")
	}

	err = conn.sendMD(conn.meta)
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not send metadata to peer")
	}

	conn.peer.meta, err = conn.recvMD()
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not recv metadata from peer")
	}

	// FIXME(sbinet): if security mechanism does not define a client/server
	// topology, enforce that p.server == p.peer.server == 0
	// as per:
	//  https://rfc.zeromq.org/spec:23/ZMTP/#topology

	return nil
}

func (conn *Conn) greet(server bool) error {
	var err error
	send := greeting{Version: defaultVersion}
	send.Sig.Header = sigHeader
	send.Sig.Footer = sigFooter
	kind := string(conn.sec.Type())
	if len(kind) > len(send.Mechanism) {
		return errSecMech
	}
	copy(send.Mechanism[:], kind)

	err = send.write(conn.rw)
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not send greeting")
	}

	var recv greeting
	err = recv.read(conn.rw)
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not recv greeting")
	}

	peerKind := asString(recv.Mechanism[:])
	if peerKind != kind {
		return errBadSec
	}

	conn.peer.server, err = asBool(recv.Server)
	if err != nil {
		return errors.Wrapf(err, "zmq4: could not get peer server flag")
	}

	return nil
}

func (c *Conn) sendMD(appMD map[string]string) error {
	md, err := c.metadata(appMD)
	if err != nil {
		return err
	}
	return c.SendCmd(CmdReady, md)
}

// Metadata returns the ZMTP-serialized metadata for this connection.
func (c *Conn) Metadata() ([]byte, error) {
	return c.metadata(c.meta)
}

func (c *Conn) metadata(appMD map[string]string) ([]byte, error) {
	buf := new(bytes.Buffer)
	keys := make(map[string]struct{})

	for k, v := range appMD {
		if len(k) == 0 {
			return nil, errEmptyAppMDKey
		}

		key := strings.ToLower(k)
		if _, dup := keys[key]; dup {
			return nil, errDupAppMDKey
		}

		keys[key] = struct{}{}
		if _, err := io.Copy(buf, Property{K: "X-" + key, V: v}); err != nil {
			return nil, err
		}
	}

	if _, err := io.Copy(buf, Property{K: sysSockType, V: string(c.typ)}); err != nil {
		return nil, err
	}
	if _, err := io.Copy(buf, Property{K: sysSockID, V: c.id.String()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Conn) recvMD() (map[string]string, error) {
	msg := c.read()
	if msg.err != nil {
		return nil, msg.err
	}

	if !msg.isCmd() {
		return nil, ErrBadFrame
	}

	var cmd Cmd
	err := cmd.unmarshalZMTP(msg.Frames[0])
	if err != nil {
		return nil, err
	}

	if cmd.Name != CmdReady {
		return nil, ErrBadCmd
	}

	sysMetadata := make(map[string]string)
	appMetadata := make(map[string]string)
	i := 0
	for i < len(cmd.Body) {
		var kv Property
		n, err := kv.Write(cmd.Body[i:])
		if err != nil {
			return nil, err
		}
		i += n

		name := strings.Title(kv.K)
		if strings.HasPrefix(name, "X-") {
			appMetadata[name] = kv.V
		} else {
			sysMetadata[name] = kv.V
			appMetadata[name] = kv.V
		}
	}

	peer := SocketType(sysMetadata[sysSockType])
	if !peer.IsCompatible(c.typ) {
		return nil, errors.Errorf("zmq4: peer=%q not compatible with %q", peer, c.typ)
	}
	return appMetadata, nil
}

// SendCmd sends a ZMTP command over the wire.
func (c *Conn) SendCmd(name string, body []byte) error {
	cmd := Cmd{Name: name, Body: body}
	buf, err := cmd.marshalZMTP()
	if err != nil {
		return err
	}
	return c.send(true, buf, 0)
}

// SendMsg sends a ZMTP message over the wire.
func (c *Conn) SendMsg(msg Msg) error {
	nframes := len(msg.Frames)
	for i, frame := range msg.Frames {
		var flag byte
		if i < nframes-1 {
			flag ^= hasMoreBitFlag
		}
		err := c.send(false, frame, flag)
		if err != nil {
			return errors.Wrapf(err, "zmq4: error sending frame %d/%d", i+1, nframes)
		}
	}
	return nil
}

// RecvMsg receives a ZMTP message from the wire.
func (c *Conn) RecvMsg() (Msg, error) {
	msg := c.read()
	if msg.err != nil {
		return msg, errors.WithStack(msg.err)
	}

	if !msg.isCmd() {
		return msg, nil
	}

	switch len(msg.Frames) {
	case 0:
		msg.err = errors.Errorf("zmq4: empty command")
		return msg, msg.err
	case 1:
		// ok
	default:
		msg.err = errors.Errorf("zmq4: invalid length command")
		return msg, msg.err
	}

	var cmd Cmd
	msg.err = cmd.unmarshalZMTP(msg.Frames[0])
	if msg.err != nil {
		return msg, errors.WithStack(msg.err)
	}

	switch cmd.Name {
	case CmdPing:
		// send back a PONG immediately.
		msg.err = c.SendCmd(CmdPong, nil)
		if msg.err != nil {
			return msg, msg.err
		}
	}

	switch len(cmd.Body) {
	case 0:
		msg.Frames = nil
	default:
		msg.Frames = msg.Frames[:1]
		msg.Frames[0] = cmd.Body
	}
	return msg, nil
}

func (c *Conn) RecvCmd() (Cmd, error) {
	var cmd Cmd
	msg := c.read()
	if msg.err != nil {
		return cmd, errors.WithStack(msg.err)
	}

	if !msg.isCmd() {
		return cmd, ErrBadFrame
	}

	switch len(msg.Frames) {
	case 0:
		msg.err = errors.Errorf("zmq4: empty command")
		return cmd, msg.err
	case 1:
		// ok
	default:
		msg.err = errors.Errorf("zmq4: invalid length command")
		return cmd, msg.err
	}

	err := cmd.unmarshalZMTP(msg.Frames[0])
	if err != nil {
		return cmd, errors.WithStack(err)
	}

	return cmd, nil
}

func (c *Conn) send(isCommand bool, body []byte, flag byte) error {
	// Long flag
	size := len(body)
	isLong := size > 255
	if isLong {
		flag ^= isLongBitFlag
	}

	if isCommand {
		flag ^= isCommandBitFlag
	}

	var (
		hdr = [8 + 1]byte{flag}
		hsz int
	)

	// Write out the message itself
	if isLong {
		hsz = 9
		binary.BigEndian.PutUint64(hdr[1:], uint64(size))
	} else {
		hsz = 2
		hdr[1] = uint8(size)
	}
	if _, err := c.rw.Write(hdr[:hsz]); err != nil {
		return err
	}

	if _, err := c.sec.Encrypt(c.rw, body); err != nil {
		return err
	}

	return nil
}

// read returns the isCommand flag, the body of the message, and optionally an error
func (c *Conn) read() Msg {
	var (
		header  [2]byte
		longHdr [8]byte
		msg     Msg

		hasMore = true
		isCmd   = false
	)

	for hasMore {

		// Read out the header
		_, msg.err = io.ReadFull(c.rw, header[:])
		if msg.err != nil {
			return msg
		}

		fl := flag(header[0])

		hasMore = fl.hasMore()
		isCmd = isCmd || fl.isCommand()

		// Determine the actual length of the body
		size := uint64(header[1])
		if fl.isLong() {
			// We read 2 bytes of the header already
			// In case of a long message, the length is bytes 2-8 of the header
			// We already have the first byte, so assign it, and then read the rest
			longHdr[0] = header[1]

			_, msg.err = io.ReadFull(c.rw, longHdr[1:])
			if msg.err != nil {
				return msg
			}

			size = binary.BigEndian.Uint64(longHdr[:])
		}

		if size > uint64(maxInt64) {
			msg.err = errOverflow
			return msg
		}

		body := make([]byte, size)
		_, msg.err = io.ReadFull(c.rw, body)
		if msg.err != nil {
			return msg
		}

		// fast path for NULL security: we bypass the bytes.Buffer allocation.
		if c.sec.Type() == NullSecurity {
			msg.Frames = append(msg.Frames, body)
			continue
		}

		buf := new(bytes.Buffer)
		if _, msg.err = c.sec.Decrypt(buf, body); msg.err != nil {
			return msg
		}
		msg.Frames = append(msg.Frames, buf.Bytes())
	}
	if isCmd {
		msg.Type = CmdMsg
	}
	return msg
}

func (conn *Conn) subscribe(msg Msg) {
	conn.mu.Lock()
	v := msg.Frames[0]
	k := string(v[1:])
	switch v[0] {
	case 0:
		delete(conn.topics, k)
	case 1:
		conn.topics[k] = struct{}{}
	}
	conn.mu.Unlock()
}

func (conn *Conn) subscribed(topic string) bool {
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	for k := range conn.topics {
		switch {
		case k == "":
			// subscribed to everything
			return true
		case strings.HasPrefix(topic, k):
			return true
		}
	}
	return false
}
