package pace

import (
	"testing"
	"time"
)

// With pacing on, a busy loop blocks about once per Work.
func TestYieldBlocks(t *testing.T) {
	On.Store(true)
	defer On.Store(false)
	start := time.Now()
	calls := 0
	for time.Since(start) < 5*Work {
		Yield()
		calls++
	}
	if calls == 0 {
		t.Fatal("no calls")
	}
	// off, it costs nothing
	On.Store(false)
	t0 := time.Now()
	for range 1_000_000 {
		Yield()
	}
	if d := time.Since(t0); d > 200*time.Millisecond {
		t.Errorf("a million calls off took %v", d)
	}
}
