package zmq4

import (
	"context"
	"testing"
	"time"
)

func TestPushTimeout(t *testing.T) {
	ep := "ipc://@push_timeout_test"
	push := NewPush(context.Background(), WithTimeout(1*time.Second))
	defer push.Close()
	if err := push.Listen(ep); err != nil {
		t.FailNow()
	}

	pull := NewPull(context.Background())
	defer pull.Close()
	if err := pull.Dial(ep); err != nil {
		t.FailNow()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			// The ctx limits overall time of execution
			// If it gets canceled, that meains tests failed
			// as write to socket did not genereate timeout error
			t.FailNow()
		default:
		}

		err := push.Send(NewMsgString("test string"))
		if err == nil {
			continue
		}
		if err != context.DeadlineExceeded {
			t.FailNow()
		}
		break
	}

}
