// Copyright 2019 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"reflect"
	"testing"
)

func makeMsg(i int) Msg {
	return NewMsgString(string(i))
}

func TestQueue(t *testing.T) {
	q := NewQueue()
	if q.Len() != 0 {
		t.Fatal("queue should be empty")
	}
	if _, exists := q.Peek(); exists {
		t.Fatal("Queue should be empty")
	}

	q.Push(makeMsg(1))
	if q.Len() != 1 {
		t.Fatal("queue should contain 1 element")
	}
	msg, ok := q.Peek()
	if !ok || !reflect.DeepEqual(msg, makeMsg(1)) {
		t.Fatal("unexpected value in queue")
	}

	q.Push(makeMsg(2))
	if q.Len() != 2 {
		t.Fatal("queue should contain 2 elements")
	}
	msg, ok = q.Peek()
	if !ok || !reflect.DeepEqual(msg, makeMsg(1)) {
		t.Fatal("unexpected value in queue")
	}

	q.Pop()
	if q.Len() != 1 {
		t.Fatal("queue should contain 1 element")
	}
	msg, ok = q.Peek()
	if !ok || !reflect.DeepEqual(msg, makeMsg(2)) {
		t.Fatal("unexpected value in queue")
	}

	q.Pop()
	if q.Len() != 0 {
		t.Fatal("queue should be empty")
	}

	q.Push(makeMsg(1))
	q.Push(makeMsg(2))
	q.Init()
	if q.Len() != 0 {
		t.Fatal("queue should be empty")
	}
}

func TestQueueNewInnerList(t *testing.T) {
	q := NewQueue()

	for i := 1; i <= innerCap; i++ {
		q.Push(makeMsg(i))
	}

	if q.Len() != innerCap {
		t.Fatalf("queue should contain %d elements", innerCap)
	}

	// next push will create a new inner slice
	q.Push(makeMsg(innerCap + 1))
	if q.Len() != innerCap+1 {
		t.Fatalf("queue should contain %d elements", innerCap+1)
	}
	msg, ok := q.Peek()
	if !ok || !reflect.DeepEqual(msg, makeMsg(1)) {
		t.Fatal("unexpected value in queue")
	}

	q.Pop()
	if q.Len() != innerCap {
		t.Fatalf("queue should contain %d elements", innerCap)
	}
	msg, ok = q.Peek()
	if !ok || !reflect.DeepEqual(msg, makeMsg(2)) {
		t.Fatal("unexpected value in queue")
	}

	q.Push(makeMsg(innerCap + 1))
	q.Init()
	if q.Len() != 0 {
		t.Fatal("queue should be empty")
	}
}
