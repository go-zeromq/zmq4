// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"log"
	"time"
)

// Option configures some aspect of a ZeroMQ socket.
// (e.g. SocketIdentity, Security, ...)
type Option func(s *socket)

// WithID configures a ZeroMQ socket identity.
func WithID(id SocketIdentity) Option {
	return func(s *socket) {
		s.id = id
	}
}

// WithSecurity configures a ZeroMQ socket to use the given security mechanism.
// If the security mechanims is nil, the NULL mechanism is used.
func WithSecurity(sec Security) Option {
	return func(s *socket) {
		s.sec = sec
	}
}

// WithDialerRetry configures the time to wait before two failed attempts
// at dialing an endpoint.
func WithDialerRetry(retry time.Duration) Option {
	return func(s *socket) {
		s.retry = retry
	}
}

// WithDialerTimeout sets the maximum amount of time a dial will wait
// for a connect to complete.
func WithDialerTimeout(timeout time.Duration) Option {
	return func(s *socket) {
		s.dialer.Timeout = timeout
	}
}

// WithLogger sets a dedicated log.Logger for the socket.
func WithLogger(msg *log.Logger) Option {
	return func(s *socket) {
		s.log = msg
	}
}

// WithDialerMaxRetries configures the maximum number of retries
// when dialing an endpoint (-1 means infinite retries).
func WithDialerMaxRetries(maxRetries int) Option {
	return func(s *socket) {
		s.maxRetries = maxRetries
	}
}

// WithAutomaticReconnect allows to configure a socket to automatically
// reconnect on connection loss.
func WithAutomaticReconnect(automaticReconnect bool) Option {
	return func(s *socket) {
		s.autoReconnect = automaticReconnect
	}
}

/*
// TODO(sbinet)

func WithIOThreads(threads int) Option {
	return nil
}

func WithSendBufferSize(size int) Option {
	return nil
}

func WithRecvBufferSize(size int) Option {
	return nil
}
*/

const (
	OptionSubscribe   = "SUBSCRIBE"
	OptionUnsubscribe = "UNSUBSCRIBE"
	OptionHWM         = "HWM"
)
