// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"reflect"
	"testing"

	"github.com/go-zeromq/zmq4/internal/inproc"
)

func TestTransport(t *testing.T) {
	if got, want := Transports(), []string{"inproc", "ipc", "tcp", "udp"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid list of transports.\ngot= %q\nwant=%q", got, want)
	}

	err := RegisterTransport("tcp", inproc.Transport{})
	if err == nil {
		t.Fatalf("expected a duplicate-registration error")
	}
	if got, want := err.Error(), "zmq4: duplicate transport \"tcp\" (transport.netTransport)"; got != want {
		t.Fatalf("invalid duplicate registration error:\ngot= %s\nwant=%s", got, want)
	}

	err = RegisterTransport("inproc2", inproc.Transport{})
	if err != nil {
		t.Fatalf("could not register 'inproc2': %+v", err)
	}
}
