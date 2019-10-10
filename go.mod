module github.com/go-zeromq/zmq4

go 1.12

require (
	github.com/pkg/errors v0.8.1
	github.com/zeromq/goczmq v4.1.0+incompatible
	github.com/zeromq/goczmq/v4 v4.1.1
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
)

replace github.com/zeromq/goczmq/v4 => github.com/go-zeromq/goczmq/v4 v4.1.1
