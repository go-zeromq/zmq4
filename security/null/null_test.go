// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package null_test

import (
	"bytes"
	"testing"

	"github.com/go-zeromq/zmq4"
	"github.com/go-zeromq/zmq4/security/null"
)

func TestNullSecurity(t *testing.T) {
	sec := null.Security()
	if got, want := sec.Type(), zmq4.NullSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}

	err := sec.Handshake()
	if err != nil {
		t.Fatalf("error doing handshake: %v", err)
	}

	data := []byte("hello world")
	wenc := new(bytes.Buffer)
	if _, err := sec.Encrypt(wenc, data); err != nil {
		t.Fatalf("error encrypting data: %v", err)
	}

	if !bytes.Equal(wenc.Bytes(), data) {
		t.Fatalf("error encrypted data.\ngot = %q\nwant= %q\n", wenc.Bytes(), data)
	}

	wdec := new(bytes.Buffer)
	if _, err := sec.Decrypt(wdec, wenc.Bytes()); err != nil {
		t.Fatalf("error decrypting data: %v", err)
	}

	if !bytes.Equal(wdec.Bytes(), data) {
		t.Fatalf("error decrypted data.\ngot = %q\nwant= %q\n", wdec.Bytes(), data)
	}

}
