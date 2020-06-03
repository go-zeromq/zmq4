// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"fmt"
	"testing"
)

func TestSplitAddr(t *testing.T) {
	testCases := []struct {
		desc    string
		v       string
		network string
		addr    string
		err     error
	}{
		{
			desc:    "tcp wild",
			v:       "tcp://*:5000",
			network: "tcp",
			addr:    "0.0.0.0:5000",
			err:     nil,
		},
		{
			desc:    "tcp ipv4",
			v:       "tcp://127.0.0.1:6000",
			network: "tcp",
			addr:    "127.0.0.1:6000",
			err:     nil,
		},
		{
			desc:    "tcp ipv6",
			v:       "tcp://[::1]:7000",
			network: "tcp",
			addr:    "[::1]:7000",
			err:     nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			network, addr, err := splitAddr(tc.v)
			if network != tc.network {
				t.Fatalf("unexpected network: got=%v, want=%v", network, tc.network)
			}
			if addr != tc.addr {
				t.Fatalf("unexpected address: got=%q, want=%q", addr, tc.addr)
			}
			if fmt.Sprintf("%+v", err) != fmt.Sprintf("%+v", tc.err) { // nil-safe comparison errors by value
				t.Fatalf("unexpected error: got=%+v, want=%+v", err, tc.err)
			}
		})
	}
}
