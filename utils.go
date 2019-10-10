// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"golang.org/x/xerrors"
)

// splitAddr returns the triplet (network, addr, error)
func splitAddr(v string) (network, addr string, err error) {
	ep := strings.Split(v, "://")
	if len(ep) != 2 {
		err = errInvalidAddress
		return network, addr, err
	}
	var (
		host string
		port string
	)
	network = ep[0]
	switch network {
	case "tcp", "udp":
		host, port, err = net.SplitHostPort(ep[1])
		if err != nil {
			return network, addr, err
		}
		switch port {
		case "0", "*", "":
			port = "0"
		}
		switch host {
		case "", "*":
			host = "0.0.0.0"
		}
		addr = host + ":" + port
		return network, addr, err

	case "ipc":
		host = ep[1]
		port = ""
		return network, host, nil
	case "inproc":
		host = ep[1]
		return "inproc", host, nil
	default:
		err = xerrors.Errorf("zmq4: unknown protocol %q", network)
	}

	return network, addr, err
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
