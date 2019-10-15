// Copyright 2019 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"container/list"
)

const innerCap = 512

type Queue struct {
	rep *list.List
	len int
}

func NewQueue() *Queue {
	q := &Queue{list.New(), 0}
	return q
}

func (q *Queue) Len() int {
	return q.len
}

func (q *Queue) Init() {
	q.rep.Init()
	q.len = 0
}

func (q *Queue) Push(val Msg) {
	q.len++

	var i []interface{}
	elem := q.rep.Back()
	if elem != nil {
		i = elem.Value.([]interface{})
	}
	if i == nil || len(i) == innerCap {
		elem = q.rep.PushBack(make([]interface{}, 0, innerCap))
		i = elem.Value.([]interface{})
	}

	elem.Value = append(i, val)
}

func (q *Queue) Peek() (Msg, bool) {
	i := q.front()
	if i == nil {
		return Msg{}, false
	}
	return i[0].(Msg), true
}

func (q *Queue) Pop() {
	elem := q.rep.Front()
	if elem == nil {
		panic("attempting to Pop on an empty Queue")
	}

	q.len--
	i := elem.Value.([]interface{})
	i[0] = nil // remove ref to poped element
	i = i[1:]
	if len(i) == 0 {
		q.rep.Remove(elem)
	} else {
		elem.Value = i
	}
}

func (q *Queue) front() []interface{} {
	elem := q.rep.Front()
	if elem == nil {
		return nil
	}
	return elem.Value.([]interface{})
}
