package zmq4

import "testing"

func TestSplitAddr(t *testing.T) {
	testCases := []struct {
		desc    string
		v       string
		network string
		addr    string
		err     error
	}{
		{
			desc:    "tcp wild",
			v:       "tcp://*:5000",
			network: "tcp",
			addr:    "0.0.0.0:5000",
			err:     nil,
		},
		{
			desc:    "tcp ipv4",
			v:       "tcp://127.0.0.1:6000",
			network: "tcp",
			addr:    "127.0.0.1:6000",
			err:     nil,
		},
		{
			desc:    "tcp ipv6",
			v:       "tcp://[::1]:7000",
			network: "tcp",
			addr:    "[::1]:7000",
			err:     nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			network, addr, err := splitAddr(tC.v)
			if network != tC.network {
				t.Fatalf("unexpected network: %v", addr)
			}
			if addr != tC.addr {
				t.Fatalf("unexpected address: %v", addr)
			}
			if err != tC.err {
				t.Fatalf("unexpected error: %+v", err)
			}
		})
	}
}
