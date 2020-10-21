// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"fmt"
	"sort"
	"sync"

	"github.com/go-zeromq/zmq4/internal/inproc"
	"github.com/go-zeromq/zmq4/transport"
)

// Transports returns the sorted list of currently registered transports.
func Transports() []string {
	return drivers.names()
}

// RegisterTransport registers a new transport with the zmq4 package.
func RegisterTransport(name string, trans transport.Transport) error {
	return drivers.add(name, trans)
}

type transports struct {
	sync.RWMutex
	db map[string]transport.Transport
}

func (ts *transports) get(name string) (transport.Transport, bool) {
	ts.RLock()
	defer ts.RUnlock()

	v, ok := ts.db[name]
	return v, ok
}

func (ts *transports) add(name string, trans transport.Transport) error {
	ts.Lock()
	defer ts.Unlock()

	if old, dup := ts.db[name]; dup {
		return fmt.Errorf("zmq4: duplicate transport %q (%T)", name, old)
	}

	ts.db[name] = trans
	return nil
}

func (ts *transports) names() []string {
	ts.RLock()
	defer ts.RUnlock()

	o := make([]string, 0, len(ts.db))
	for k := range ts.db {
		o = append(o, k)
	}
	sort.Strings(o)
	return o
}

var drivers = transports{
	db: make(map[string]transport.Transport),
}

func init() {
	must := func(err error) {
		if err != nil {
			panic(fmt.Errorf("%+v", err))
		}
	}

	must(RegisterTransport("ipc", transport.New("unix")))
	must(RegisterTransport("tcp", transport.New("tcp")))
	must(RegisterTransport("udp", transport.New("udp")))
	must(RegisterTransport("inproc", inproc.Transport{}))
}
