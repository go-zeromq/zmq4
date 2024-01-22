// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"fmt"
	"io"
	"log"
	"net"
)

var (
	Devnull = log.New(io.Discard, "zmq4: ", 0)
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
