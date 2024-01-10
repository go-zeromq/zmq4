//go:build TestConnReaperDeadlockLoop

package zmq4_test

import (
	"log"
	"testing"
	"time"
)

// Use ```go test -tags=TestConnReaperDeadlockLoop -run ^TestConnReaperDeadlockLoop$```
func TestConnReaperDeadlockLoop(t *testing.T) {
	n := uint64(0)
	for {
		n++
		if n%100 == 0 {
			log.Printf("%d ...", n)
		}
		timer := time.AfterFunc(30*time.Second, func() {
			log.Fatalf("failed at %d!!!", n)
		})

		TestConnReaperDeadlock(t)

		timer.Stop()
	}
}
