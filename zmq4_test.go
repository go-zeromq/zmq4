// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	bkg = context.Background()
)

func must(str string, err error) string {
	if err != nil {
		panic(err)
	}
	return str
}

func EndPoint(transport string) (string, error) {
	switch transport {
	case "tcp":
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return "", err
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return "", err
		}
		defer l.Close()
		return fmt.Sprintf("tcp://%s", l.Addr()), nil
	case "ipc":
		return "ipc://tmp-" + newUUID(), nil
	case "inproc":
		return "inproc://tmp-" + newUUID(), nil
	default:
		panic("invalid transport: [" + transport + "]")
	}
}

func getTCPPort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

func cleanUp(ep string) {
	switch {
	case strings.HasPrefix(ep, "ipc://"):
		os.Remove(ep[len("ipc://"):])
	case strings.HasPrefix(ep, "inproc://"):
		os.Remove(ep[len("inproc://"):])
	}
}

func newUUID() string {
	var uuid [16]byte
	if _, err := io.ReadFull(rand.Reader, uuid[:]); err != nil {
		log.Fatalf("cannot generate random data for UUID: %v", err)
	}
	uuid[8] = uuid[8]&^0xc0 | 0x80
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
