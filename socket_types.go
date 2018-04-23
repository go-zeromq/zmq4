// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

// SocketType is a ZeroMQ socket type.
type SocketType string

const (
	Pair   SocketType = "PAIR"   // a ZMQ_PAIR socket
	Pub    SocketType = "PUB"    // a ZMQ_PUB socket
	Sub    SocketType = "SUB"    // a ZMQ_SUB socket
	Req    SocketType = "REQ"    // a ZMQ_REQ socket
	Rep    SocketType = "REP"    // a ZMQ_REP socket
	Dealer SocketType = "DEALER" // a ZMQ_DEALER socket
	Router SocketType = "ROUTER" // a ZMQ_ROUTER socket
	Pull   SocketType = "PULL"   // a ZMQ_PULL socket
	Push   SocketType = "PUSH"   // a ZMQ_PUSH socket
	XPub   SocketType = "XPUB"   // a ZMQ_XPUB socket
	XSub   SocketType = "XSUB"   // a ZMQ_XSUB socket
)

// IsCompatible checks whether two sockets are compatible and thus
// can be connected together.
// See https://rfc.zeromq.org/spec:23/ZMTP/ for more informations.
func (sck SocketType) IsCompatible(peer SocketType) bool {
	switch sck {
	case Pair:
		if peer == Pair {
			return true
		}
	case Pub:
		switch peer {
		case Sub, XSub:
			return true
		}
	case Sub:
		switch peer {
		case Pub, XPub:
			return true
		}
	case Req:
		switch peer {
		case Rep, Router:
			return true
		}
	case Rep:
		switch peer {
		case Req, Dealer:
			return true
		}
	case Dealer:
		switch peer {
		case Rep, Dealer, Router:
			return true
		}
	case Router:
		switch peer {
		case Req, Dealer, Router:
			return true
		}
	case Pull:
		switch peer {
		case Push:
			return true
		}
	case Push:
		switch peer {
		case Pull:
			return true
		}
	case XPub:
		switch peer {
		case Sub, XSub:
			return true
		}
	case XSub:
		switch peer {
		case Pub, XPub:
			return true
		}
	default:
		panic("unknown socket-type: \"" + string(sck) + "\"")
	}

	return false
}

// SocketIdentity is the ZMTP metadata socket identity.
// See:
//  https://rfc.zeromq.org/spec:23/ZMTP/.
type SocketIdentity []byte

func (id SocketIdentity) String() string {
	n := len(id)
	if n > 255 { // ZMTP identities are: 0*255OCTET
		n = 255
	}
	return string(id[:n])
}
