// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"strings"
)

// splitAddr returns the triplet (network, addr, error)
func splitAddr(v string) (network, addr string, err error) {
	ep := strings.Split(v, "://")
	if len(ep) != 2 {
		err = errInvalidAddress
		return network, addr, err
	}
	network = ep[0]

	trans, ok := drivers.get(network)
	if !ok {
		err = fmt.Errorf("zmq4: unknown transport %q", network)
		return network, addr, err
	}

	addr, err = trans.Addr(ep[1])
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
