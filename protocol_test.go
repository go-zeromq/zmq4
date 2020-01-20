// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"io"
	"testing"

	"golang.org/x/xerrors"
)

func TestGreeting(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{
			name: "valid",
			data: func() []byte {
				w := new(bytes.Buffer)
				g := greeting{
					Version: defaultVersion,
				}
				g.Sig.Header = sigHeader
				g.Sig.Footer = sigFooter
				err := g.write(w)
				if err != nil {
					t.Fatalf("could not marshal greeting: %+v", err)
				}
				return w.Bytes()
			}(),
		},
		{
			name: "empty-buffer",
			data: nil,
			want: xerrors.Errorf("could not read ZMTP greeting: %w", io.EOF),
		},
		{
			name: "unexpected-EOF",
			data: make([]byte, 1),
			want: xerrors.Errorf("could not read ZMTP greeting: %w", io.ErrUnexpectedEOF),
		},
		{
			name: "invalid-header",
			data: func() []byte {
				w := new(bytes.Buffer)
				g := greeting{
					Version: defaultVersion,
				}
				g.Sig.Header = sigFooter // err
				g.Sig.Footer = sigFooter
				err := g.write(w)
				if err != nil {
					t.Fatalf("could not marshal greeting: %+v", err)
				}
				return w.Bytes()
			}(),
			want: xerrors.Errorf("invalid ZMTP signature header: %w", errGreeting),
		},
		{
			name: "invalid-footer",
			data: func() []byte {
				w := new(bytes.Buffer)
				g := greeting{
					Version: defaultVersion,
				}
				g.Sig.Header = sigHeader
				g.Sig.Footer = sigHeader // err
				err := g.write(w)
				if err != nil {
					t.Fatalf("could not marshal greeting: %+v", err)
				}
				return w.Bytes()
			}(),
			want: xerrors.Errorf("invalid ZMTP signature footer: %w", errGreeting),
		},
		{
			name: "higher-major-version",
			data: func() []byte {
				w := new(bytes.Buffer)
				g := greeting{
					Version: [2]uint8{defaultVersion[0] + 1, defaultVersion[1]},
				}
				g.Sig.Header = sigHeader
				g.Sig.Footer = sigFooter
				err := g.write(w)
				if err != nil {
					t.Fatalf("could not marshal greeting: %+v", err)
				}
				return w.Bytes()
			}(),
		},
		{
			name: "higher-minor-version",
			data: func() []byte {
				w := new(bytes.Buffer)
				g := greeting{
					Version: [2]uint8{defaultVersion[0], defaultVersion[1] + 1},
				}
				g.Sig.Header = sigHeader
				g.Sig.Footer = sigFooter
				err := g.write(w)
				if err != nil {
					t.Fatalf("could not marshal greeting: %+v", err)
				}
				return w.Bytes()
			}(),
		},
		{
			name: "smaller-major-version", // FIXME(sbinet): adapt for when/if we support multiple ZMTP versions
			data: func() []byte {
				w := new(bytes.Buffer)
				g := greeting{
					Version: [2]uint8{defaultVersion[0] - 1, defaultVersion[1]},
				}
				g.Sig.Header = sigHeader
				g.Sig.Footer = sigFooter
				err := g.write(w)
				if err != nil {
					t.Fatalf("could not marshal greeting: %+v", err)
				}
				return w.Bytes()
			}(),
			want: xerrors.Errorf("invalid ZMTP version (got=%v, want=%v): %w",
				[2]uint8{defaultVersion[0] - 1, defaultVersion[1]},
				defaultVersion,
				errGreeting,
			),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				g greeting
				r = bytes.NewReader(tc.data)
			)

			err := g.read(r)
			switch {
			case err == nil && tc.want == nil:
				// ok
			case err == nil && tc.want != nil:
				t.Fatalf("expected an error (%s)", tc.want)
			case err != nil && tc.want == nil:
				t.Fatalf("could not read ZMTP greeting: %+v", err)
			case err != nil && tc.want != nil:
				if got, want := err.Error(), tc.want.Error(); got != want {
					t.Fatalf("invalid ZMTP greeting error:\ngot= %+v\nwant=%+v\n",
						got, want,
					)
				}
			}

		})
	}
}
