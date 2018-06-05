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
			endpoint: must(EndPoint("tcp")),
			push:     zmq4.NewCPush(bkg),
			pull:     zmq4.NewPull(bkg),
		},
		{
			name:     "tcp-push-cpull",
			endpoint: must(EndPoint("tcp")),
			push:     zmq4.NewPush(bkg),
			pull:     zmq4.NewCPull(bkg),
		},
		{
			name:     "tcp-cpush-cpull",
			endpoint: must(EndPoint("tcp")),
			push:     zmq4.NewCPush(bkg),
			pull:     zmq4.NewCPull(bkg),
		},
		{
			name:     "ipc-cpush-pull",
			endpoint: "ipc://ipc-cpush-pull",
			push:     zmq4.NewCPush(bkg),
			pull:     zmq4.NewPull(bkg),
		},
		{
			name:     "ipc-push-cpull",
			endpoint: "ipc://ipc-push-cpull",
			push:     zmq4.NewPush(bkg),
			pull:     zmq4.NewCPull(bkg),
		},
		{
			name:     "ipc-cpush-cpull",
			endpoint: "ipc://ipc-cpush-cpull",
			push:     zmq4.NewCPush(bkg),
			pull:     zmq4.NewCPull(bkg),
		},
		//{
		//	name:     "udp-cpush-cpull",
		//	endpoint: "udp://127.0.0.1:55555",
		//	push:     zmq4.NewCPush(),
		//	pull:     zmq4.NewCPull(),
		//},
	}

	creqreps = []testCaseReqRep{
		{
			name:     "tcp-creq-rep",
			endpoint: must(EndPoint("tcp")),
			req:      zmq4.NewCReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "tcp-req-crep",
			endpoint: must(EndPoint("tcp")),
			req:      zmq4.NewReq(bkg),
			rep:      zmq4.NewCRep(bkg),
		},
		{
			name:     "tcp-creq-crep",
			endpoint: must(EndPoint("tcp")),
			req:      zmq4.NewCReq(bkg),
			rep:      zmq4.NewCRep(bkg),
		},
		{
			name:     "ipc-creq-rep",
			endpoint: "ipc://ipc-creq-rep",
			req:      zmq4.NewCReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "ipc-req-crep",
			endpoint: "ipc://ipc-req-crep",
			req:      zmq4.NewReq(bkg),
			rep:      zmq4.NewCRep(bkg),
		},
		{
			name:     "ipc-creq-crep",
			endpoint: "ipc://ipc-creq-crep",
			req:      zmq4.NewCReq(bkg),
			rep:      zmq4.NewCRep(bkg),
		},
	}

	cpubsubs = []testCasePubSub{
		{
			name:     "tcp-cpub-sub",
			endpoint: must(EndPoint("tcp")),
			pub:      zmq4.NewCPub(bkg),
			sub0:     zmq4.NewSub(bkg),
			sub1:     zmq4.NewSub(bkg),
			sub2:     zmq4.NewSub(bkg),
		},
		{
			name:     "tcp-pub-csub",
			endpoint: must(EndPoint("tcp")),
			pub:      zmq4.NewPub(bkg),
			sub0:     zmq4.NewCSub(bkg),
			sub1:     zmq4.NewCSub(bkg),
			sub2:     zmq4.NewCSub(bkg),
		},
		{
			name:     "tcp-cpub-csub",
			endpoint: must(EndPoint("tcp")),
			pub:      zmq4.NewCPub(bkg),
			sub0:     zmq4.NewCSub(bkg),
			sub1:     zmq4.NewCSub(bkg),
			sub2:     zmq4.NewCSub(bkg),
		},
		{
			name:     "ipc-cpub-sub",
			endpoint: "ipc://ipc-cpub-sub",
			pub:      zmq4.NewCPub(bkg),
			sub0:     zmq4.NewSub(bkg),
			sub1:     zmq4.NewSub(bkg),
			sub2:     zmq4.NewSub(bkg),
		},
		{
			name:     "ipc-pub-csub",
			endpoint: "ipc://ipc-pub-csub",
			pub:      zmq4.NewPub(bkg),
			sub0:     zmq4.NewCSub(bkg),
			sub1:     zmq4.NewCSub(bkg),
			sub2:     zmq4.NewCSub(bkg),
		},
		{
			name:     "ipc-cpub-csub",
			endpoint: "ipc://ipc-cpub-csub",
			pub:      zmq4.NewCPub(bkg),
			sub0:     zmq4.NewCSub(bkg),
			sub1:     zmq4.NewCSub(bkg),
			sub2:     zmq4.NewCSub(bkg),
		},
	}
)

func init() {
	pushpulls = append(pushpulls, cpushpulls...)
	reqreps = append(reqreps, creqreps...)
	pubsubs = append(pubsubs, cpubsubs...)
}
