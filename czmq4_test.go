// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build czmq4

package zmq4_test

import (
	"github.com/go-zeromq/zmq4"
)

var (
	cpushpulls = []testCasePushPull{
		{
			name:     "tcp-cpush-pull",
			endpoint: "tcp://127.0.0.1:55552",
			push:     NewCPush(bkg),
			pull:     zmq4.NewPull(bkg),
		},
		{
			name:     "tcp-push-cpull",
			endpoint: "tcp://127.0.0.1:55553",
			push:     zmq4.NewPush(bkg),
			pull:     NewCPull(bkg),
		},
		{
			name:     "tcp-cpush-cpull",
			endpoint: "tcp://127.0.0.1:55554",
			push:     NewCPush(bkg),
			pull:     NewCPull(bkg),
		},
		{
			name:     "ipc-cpush-pull",
			endpoint: "ipc://ipc-cpush-pull",
			push:     NewCPush(bkg),
			pull:     zmq4.NewPull(bkg),
		},
		{
			name:     "ipc-push-cpull",
			endpoint: "ipc://ipc-push-cpull",
			push:     zmq4.NewPush(bkg),
			pull:     NewCPull(bkg),
		},
		{
			name:     "ipc-cpush-cpull",
			endpoint: "ipc://ipc-cpush-cpull",
			push:     NewCPush(bkg),
			pull:     NewCPull(bkg),
		},
		//{
		//	name:     "udp-cpush-cpull",
		//	endpoint: "udp://127.0.0.1:55555",
		//	push:     NewCPush(),
		//	pull:     NewCPull(),
		//},
	}
)

func init() {
	pushpulls = append(pushpulls, cpushpulls...)
}
