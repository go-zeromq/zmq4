// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

var (
	pubsubs = []testCasePubSub{
		{
			name:     "tcp-pub-sub",
			endpoint: must(EndPoint("tcp")),
			pub:      zmq4.NewPub(bkg),
			sub0:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub0"))),
			sub1:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub1"))),
			sub2:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub2"))),
		},
		{
			name:     "ipc-pub-sub",
			endpoint: "ipc://ipc-pub-sub",
			pub:      zmq4.NewPub(bkg),
			sub0:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub0"))),
			sub1:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub1"))),
			sub2:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub2"))),
		},
		{
			name:     "inproc-pub-sub",
			endpoint: "inproc://inproc-pub-sub",
			pub:      zmq4.NewPub(bkg),
			sub0:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub0"))),
			sub1:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub1"))),
			sub2:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub2"))),
		},
	}
)

type testCasePubSub struct {
	name     string
	skip     bool
	endpoint string
	pub      zmq4.Socket
	sub0     zmq4.Socket
	sub1     zmq4.Socket
	sub2     zmq4.Socket
}

func TestNotBlockingSendOnPub(t *testing.T) {

	pub := zmq4.NewPub(context.Background())
	defer pub.Close()

	err := pub.Listen(must(EndPoint("tcp")))
	if err != nil {
		t.Fatalf("could not listen on end point: %+v", err)
	}

	errc := make(chan error)
	go func() {
		errc <- pub.Send(zmq4.NewMsg([]byte("blocked?")))
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("pub socket should not block!")
	case err := <-errc:
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
	}
}

func TestPubSub(t *testing.T) {
	var (
		topics      = []string{"", "MSG", "msg"}
		wantNumMsgs = []int{3, 1, 1}
		msg0        = zmq4.NewMsgString("anything")
		msg1        = zmq4.NewMsgString("MSG 1")
		msg2        = zmq4.NewMsgString("msg 2")
		msgs        = [][]zmq4.Msg{
			0: {msg0, msg1, msg2},
			1: {msg1},
			2: {msg2},
		}
	)

	for i := range pubsubs {
		tc := pubsubs[i]
		t.Run(tc.name, func(t *testing.T) {
			defer tc.pub.Close()
			defer tc.sub0.Close()
			defer tc.sub1.Close()
			defer tc.sub2.Close()

			ep := tc.endpoint
			cleanUp(ep)

			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()

			ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
			defer timeout()

			nmsgs := []int{0, 0, 0}
			subs := []zmq4.Socket{tc.sub0, tc.sub1, tc.sub2}

			var wg1 sync.WaitGroup
			var wg2 sync.WaitGroup
			wg1.Add(len(subs))
			wg2.Add(len(subs))

			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := tc.pub.Listen(ep)
				if err != nil {
					return xerrors.Errorf("could not listen: %w", err)
				}

				if addr := tc.pub.Addr(); addr == nil {
					return xerrors.Errorf("listener with nil Addr")
				}

				wg1.Wait()
				wg2.Wait()

				time.Sleep(1 * time.Second)

				for _, msg := range msgs[0] {
					err = tc.pub.Send(msg)
					if err != nil {
						return xerrors.Errorf("could not send message %v: %w", msg, err)
					}
				}

				return err
			})

			for isub := range subs {
				func(isub int, sub zmq4.Socket) {
					grp.Go(func() error {
						var err error
						err = sub.Dial(ep)
						if err != nil {
							return xerrors.Errorf("could not dial: %w", err)
						}

						if addr := sub.Addr(); addr != nil {
							return xerrors.Errorf("dialer with non-nil Addr")
						}

						wg1.Done()
						wg1.Wait()

						err = sub.SetOption(zmq4.OptionSubscribe, topics[isub])
						if err != nil {
							return xerrors.Errorf("could not subscribe to topic %q: %w", topics[isub], err)
						}

						wg2.Done()
						wg2.Wait()

						msgs := msgs[isub]
						for imsg, want := range msgs {
							msg, err := sub.Recv()
							if err != nil {
								return xerrors.Errorf("could not recv message %v: %w", want, err)
							}
							if !reflect.DeepEqual(msg, want) {
								return xerrors.Errorf("sub[%d][msg=%d]: got = %v, want= %v", isub, imsg, msg, want)
							}
							nmsgs[isub]++
						}

						return err
					})
				}(isub, subs[isub])
			}

			if err := grp.Wait(); err != nil {
				t.Fatalf("error: %+v", err)
			}

			for i, want := range wantNumMsgs {
				if want != nmsgs[i] {
					t.Errorf("sub[%d]: got %d messages, want %d msgs=%v", i, nmsgs[i], want, nmsgs)
				}
			}
		})
	}
}

// TestPubSubClosedSub ensures that publishers do not return errors even after a subscriber is closed/disconnected.
func TestPubSubClosedSub(t *testing.T) {
	ep := must(EndPoint("tcp"))
	topic := "msg"
	msg := zmq4.NewMsgString("msg")

	bkg := context.Background()
	ctx, timeout := context.WithTimeout(bkg, 20*time.Second)
	defer timeout()

	pub := zmq4.NewPub(ctx)
	defer pub.Close()

	subCtx, cancelSub := context.WithCancel(ctx)
	sub := zmq4.NewSub(subCtx)

	subReady := make(chan struct{})
	subClosed := make(chan struct{})

	const nmsgs = 100 // the number of messages do not matter

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		err := pub.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen on end point: %+v", err)
		}

		<-subReady
		time.Sleep(time.Second * 1)

		for i := 0; i < nmsgs; i++ {
			if err := pub.Send(msg); err != nil {
				return xerrors.Errorf("could not send message %v: %w", msg, err)
			}
		}

		cancelSub()
		<-subClosed
		time.Sleep(time.Second * 1)

		for i := 0; i < nmsgs; i++ {
			if err := pub.Send(msg); err != nil {
				return xerrors.Errorf("could not send message %v: %w", msg, err)
			}
		}

		return err
	})

	grp.Go(func() error {
		defer func() {
			sub.Close()
			close(subClosed)
		}()

		var err error
		err = sub.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial: %w", err)
		}

		err = sub.SetOption(zmq4.OptionSubscribe, topic)
		if err != nil {
			return xerrors.Errorf("could not subscribe to topic %q: %w", topic, err)
		}

		close(subReady)

		for {
			rmsg, err := sub.Recv()
			if subCtx.Err() == context.Canceled {
				break
			}
			if err != nil {
				return xerrors.Errorf("could not recv message: %w", err)
			}
			if !reflect.DeepEqual(rmsg, msg) {
				return xerrors.Errorf("sub: got = %v, want= %v", rmsg, msg)
			}
		}

		return err
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func TestPubSubMultiPart(t *testing.T) {
	msg := zmq4.NewMsgFrom([]byte("msgA"), []byte("msgB"), []byte("msgC"))
	pub := zmq4.NewPub(bkg)
	sub := zmq4.NewSub(bkg)

	defer pub.Close()
	defer sub.Close()

	ep := must(EndPoint("tcp"))
	cleanUp(ep)

	ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
	defer timeout()

	grp, _ := errgroup.WithContext(ctx)

	pss := newPubSubSync(1)

	grp.Go(func() error {
		var err error
		err = pub.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen on end point: %+v", err)
		}
		if addr := pub.Addr(); addr == nil {
			return xerrors.Errorf("listener with nil Addr")
		}

		pss.WaitForSubscriptions()
		time.Sleep(1 * time.Second)

		err = pub.SendMulti(msg)
		if err != nil {
			return xerrors.Errorf("could not send message %v: %w", msg, err)
		}

		return nil
	})

	grp.Go(func() error {
		var err error
		err = sub.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial: %w", err)
		}

		if addr := sub.Addr(); addr != nil {
			return xerrors.Errorf("dialer with non-nil Addr")
		}

		pss.DialComplete()
		pss.WaitForDialers()

		err = sub.SetOption(zmq4.OptionSubscribe, "msg")
		if err != nil {
			return xerrors.Errorf("could not subscribe to topic: %w", err)
		}

		pss.SubscriptionComplete()
		pss.WaitForSubscriptions()

		newMsg, err := sub.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv message %v: %w", msg, err)
		}
		if !reflect.DeepEqual(newMsg, msg) {
			return xerrors.Errorf("got = %v, want= %v", newMsg, msg)
		}
		return err
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func TestTopics(t *testing.T) {
	ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
	defer timeout()

	ep := must(EndPoint("tcp"))
	pub := zmq4.NewPub(ctx)
	sub0 := zmq4.NewSub(ctx, zmq4.WithID(zmq4.SocketIdentity("sub0")))
	sub1 := zmq4.NewSub(ctx, zmq4.WithID(zmq4.SocketIdentity("sub1")))
	sub2 := zmq4.NewSub(ctx, zmq4.WithID(zmq4.SocketIdentity("sub2")))
	sub3 := zmq4.NewSub(ctx, zmq4.WithID(zmq4.SocketIdentity("sub3")))
	sub4 := zmq4.NewSub(ctx, zmq4.WithID(zmq4.SocketIdentity("sub4")))
	sub5 := zmq4.NewSub(ctx, zmq4.WithID(zmq4.SocketIdentity("sub5")))

	subs := []zmq4.Socket{sub0, sub1, sub2, sub3, sub4, sub5}

	defer pub.Close()
	defer sub0.Close()
	defer sub1.Close()
	defer sub2.Close()
	defer sub3.Close()
	defer sub4.Close()
	defer sub5.Close()

	err := pub.Listen(ep)
	if err != nil {
		t.Fatalf("could not listen: %+v", err)
	}

	for isub, sub := range subs {
		topics := []string{"", "a", "b", "c", "2", "A_2"}

		err = sub.Dial(ep)
		if err != nil {
			t.Fatalf("could not dial: %+v", err)
		}

		err = sub.SetOption(zmq4.OptionSubscribe, topics[isub])
		if err != nil {
			t.Fatalf("could not subscribe to topic %q: %+v", topics[isub], err)
		}
		time.Sleep(500 * time.Millisecond)

		got := sub.(zmq4.Topics).Topics()
		want := []string{topics[isub]}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Missing or wrong topics.\ngot= %q\nwant=%q", got, want)
		}

		got = pub.(zmq4.Topics).Topics()
		if len(got) != isub+1 {
			t.Fatalf("got %d topics, want %d topics", len(got), isub+1)
		}

		want = make([]string, isub+1)
		copy(want, topics)
		sort.Strings(want)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Missing or wrong topics.\ngot= %q\nwant=%q", got, want)
		}
	}
}

// TestPubSubDeadPub ensures that subscribers can proceed even after losing connection to the publisher
func TestPubSubDeadPub(t *testing.T) {
	ep := must(EndPoint("tcp"))
	topic := "msg"
	msg := zmq4.NewMsgString("msg")

	bkg := context.Background()
	ctx, timeout := context.WithTimeout(bkg, 20*time.Second)
	defer timeout()

	pub := zmq4.NewPub(ctx)
	sub := zmq4.NewSub(ctx)

	defer sub.Close()

	subReady := make(chan struct{})
	subDoneReading := make(chan struct{})
	pubClosed := make(chan struct{})

	const nmsgs = 4 // the number of messages do not matter

	grp, _ := errgroup.WithContext(ctx)
	grp.Go(func() error {
		defer close(pubClosed)
		defer pub.Close()

		err := pub.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen on end point: %+v", err)
		}

		<-subReady
		time.Sleep(time.Second * 1)

		for i := 0; i < nmsgs; i++ {
			if err := pub.Send(msg); err != nil {
				return xerrors.Errorf("could not send message %v: %w", msg, err)
			}
		}
		<-subDoneReading

		return err
	})

	grp.Go(func() error {
		var err error
		err = sub.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial: %w", err)
		}

		err = sub.SetOption(zmq4.OptionSubscribe, topic)
		if err != nil {
			return xerrors.Errorf("could not subscribe to topic %q: %w", topic, err)
		}

		close(subReady)

		for i := 0; i < nmsgs; i++ {
			rmsg, err := sub.Recv()
			if err != nil {
				return xerrors.Errorf("could not recv message: %w", err)
			}
			if !reflect.DeepEqual(rmsg, msg) {
				return xerrors.Errorf("sub: got = %v, want= %v", rmsg, msg)
			}
		}

		close(subDoneReading)
		<-pubClosed

		_, err = sub.Recv() // make sure we aren't deadlocked
		if err == nil {
			return xerrors.New("expected an error")
		}

		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func TestPubOptionHWM(t *testing.T) {
	topic := "msg"
	pub := zmq4.NewPub(bkg)

	subCtx, subCancel := context.WithCancel(bkg)
	sub := zmq4.NewSub(subCtx)

	defer pub.Close()
	defer sub.Close()

	msgCount := 100
	hwm := 10
	if err := pub.SetOption(zmq4.OptionHWM, hwm); err != nil {
		t.Fatalf("unable to set HWM")
	}

	ep := must(EndPoint("tcp"))
	cleanUp(ep)

	ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
	defer timeout()

	grp, _ := errgroup.WithContext(ctx)
	pss := newPubSubSync(1)

	grp.Go(func() error {
		var err error
		err = pub.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen on end point: %+v", err)
		}

		pss.WaitForSubscriptions()

		for i := 1; i <= msgCount; i++ {
			msg := zmq4.NewMsgFrom([]byte("msg"), []byte(string(i)))
			err = pub.Send(msg)
			if err != nil {
				return xerrors.Errorf("error sending message. [%d] got: %v", i, err)
			}
		}

		// give the subscriber time to receive the last message
		time.Sleep(time.Second * 2)
		// Inform the subscriber that there are no more messages, otherwise it'll wait indefinitely while trying to receive dropped messages
		subCancel()
		return nil
	})

	grp.Go(func() error {
		var err error
		err = sub.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial end point: %+v", err)
		}

		pss.DialComplete()
		pss.WaitForDialers()

		err = sub.SetOption(zmq4.OptionSubscribe, topic)
		if err != nil {
			return xerrors.Errorf("could not subscribe to topic %q: %w", topic, err)
		}

		pss.SubscriptionComplete()
		pss.WaitForSubscriptions()

		time.Sleep(time.Second * 1) // slow down for a bit

		nmsgs := 0

		for i := 1; i <= msgCount; i++ {
			_, err := sub.Recv()
			if subCtx.Err() != nil {
				break
			}
			if err != nil {
				return xerrors.Errorf("could not recv message: %v", err)
			}
			nmsgs++
		}

		if nmsgs >= msgCount {
			return xerrors.Errorf("Expected dropped messages")
		}

		return err
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func BenchmarkPubSub(b *testing.B) {
	topic := "msg"
	msgs := make([][]byte, 10)
	for i := range msgs {
		msgs[i] = []byte("msg")
	}
	msg := zmq4.NewMsgFrom(msgs...)
	pub := zmq4.NewPub(bkg)
	sub := zmq4.NewSub(bkg)

	defer pub.Close()
	defer sub.Close()

	ep := must(EndPoint("tcp"))
	cleanUp(ep)

	ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
	defer timeout()

	grp, _ := errgroup.WithContext(ctx)

	msgCount := 1 << 18
	pss := newPubSubSync(1)

	grp.Go(func() error {
		var err error
		err = pub.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen on end point: %+v", err)
		}

		pss.WaitForSubscriptions()
		time.Sleep(1 * time.Second)

		for i := 0; i < msgCount; i++ {
			err = pub.SendMulti(msg)
			if err != nil {
				return xerrors.Errorf("error sending message: %v\n", err)
			}
		}

		return err
	})

	grp.Go(func() error {
		var err error
		err = sub.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial end point: %+v", err)
		}

		pss.DialComplete()
		pss.WaitForDialers()

		err = sub.SetOption(zmq4.OptionSubscribe, topic)
		if err != nil {
			return xerrors.Errorf("could not subscribe to topic %q: %w", topic, err)
		}

		pss.SubscriptionComplete()
		pss.WaitForSubscriptions()

		var siz int
		for i := 0; i < msgCount; i++ {
			msg, err := sub.Recv()
			if err != nil {
				return xerrors.Errorf("could not recv message: %v", err)
			}
			for _, frame := range msg.Frames {
				siz += len(frame)
			}
		}

		return err
	})

	if err := grp.Wait(); err != nil {
		b.Fatalf("error: %+v", err)
	}
}

type pubSubSync struct {
	wg1 sync.WaitGroup
	wg2 sync.WaitGroup
}

func newPubSubSync(nrSubs int) *pubSubSync {
	p := &pubSubSync{}
	p.wg1.Add(nrSubs)
	p.wg2.Add(nrSubs)
	return p
}

func (p *pubSubSync) DialComplete() {
	p.wg1.Done()
}

func (p *pubSubSync) WaitForDialers() {
	p.wg1.Wait()
}

func (p *pubSubSync) SubscriptionComplete() {
	p.wg2.Done()
}

func (p *pubSubSync) WaitForSubscriptions() {
	p.wg2.Wait()
}
