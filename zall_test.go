// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"io"
	"log"
)

var (
	Devnull = log.New(io.Discard, "zmq4: ", 0)
)
