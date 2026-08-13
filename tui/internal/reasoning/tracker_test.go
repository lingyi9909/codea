package reasoning

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	t time.Time
}

func (f *fakeClock) Now() time.Time { return f.t }
func (f *fakeClock) Advance(d time.Duration) {
	f.t = f.t.Add(d)
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)}
}

func TestTrackerStartActive(t *testing.T) {
	tr := NewTracker()
	b := tr.Start()
	if b.State != BlockActive {
		t.Fatalf("expected BlockActive, got %v", b.State)
	}
	if b.Index != 0 {
		t.Fatalf("expected Index=0, got %d", b.Index)
	}
	if b.StartedAt.IsZero() {
		t.Fatal("expected non-zero StartedAt")
	}
}

func TestTrackerAppendAccumulates(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("hello ")
	tr.Append("world")
	b, ok := tr.Active()
	if !ok {
		t.Fatal("expected active block")
	}
	if b.Content != "hello world" {
		t.Fatalf("expected content 'hello world', got %q", b.Content)
	}
}

func TestTrackerStreamingDeltasNotDropped(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	for _, d := range []string{"a", "b", "c", "d", "e"} {
		tr.Append(d)
	}
	tr.End()
	snap := tr.Snapshot()
	if len(snap.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(snap.Blocks))
	}
	if snap.Blocks[0].Content != "abcde" {
		t.Fatalf("expected merged content 'abcde', got %q", snap.Blocks[0].Content)
	}
}

func TestTrackerEndCompleted(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("think")
	b, ok := tr.End()
	if !ok {
		t.Fatal("expected End to succeed")
	}
	if b.State != BlockCompleted {
		t.Fatalf("expected BlockCompleted, got %v", b.State)
	}
	if b.EndedAt.IsZero() {
		t.Fatal("expected non-zero EndedAt")
	}
	if snap := tr.Snapshot(); snap.HasActive() {
		t.Fatal("expected no active block after End")
	}
}

func TestTrackerDurationCorrect(t *testing.T) {
	clk := newFakeClock()
	tr := NewTracker(WithClock(clk))
	tr.Start()
	clk.Advance(2800 * time.Millisecond)
	b, ok := tr.End()
	if !ok {
		t.Fatal("expected End to succeed")
	}
	if b.Duration != 2800*time.Millisecond {
		t.Fatalf("expected duration 2.8s, got %v", b.Duration)
	}
	if b.Duration != b.EndedAt.Sub(b.StartedAt) {
		t.Fatalf("expected Duration to equal EndedAt-StartedAt")
	}
}

func TestTrackerActiveHasZeroDuration(t *testing.T) {
	clk := newFakeClock()
	tr := NewTracker(WithClock(clk))
	tr.Start()
	clk.Advance(5 * time.Second)
	b, _ := tr.Active()
	if b.Duration != 0 {
		t.Fatalf("expected active block Duration=0, got %v", b.Duration)
	}
}

func TestTrackerMultipleBlocksOrdered(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("first")
	tr.End()
	tr.Start()
	tr.Append("second")
	tr.End()

	snap := tr.Snapshot()
	if len(snap.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(snap.Blocks))
	}
	if snap.Blocks[0].Index != 0 || snap.Blocks[0].Content != "first" || snap.Blocks[0].State != BlockCompleted {
		t.Fatalf("block 0 unexpected: %+v", snap.Blocks[0])
	}
	if snap.Blocks[1].Index != 1 || snap.Blocks[1].Content != "second" || snap.Blocks[1].State != BlockCompleted {
		t.Fatalf("block 1 unexpected: %+v", snap.Blocks[1])
	}
}

func TestTrackerStartCompletesPrevious(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("old")
	tr.Start()
	tr.Append("new")

	snap := tr.Snapshot()
	if len(snap.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(snap.Blocks))
	}
	if snap.Blocks[0].Content != "old" || snap.Blocks[0].State != BlockCompleted {
		t.Fatalf("previous block not completed cleanly: %+v", snap.Blocks[0])
	}
	if snap.Blocks[1].Content != "new" || snap.Blocks[1].State != BlockActive {
		t.Fatalf("new block not active: %+v", snap.Blocks[1])
	}
}

func TestTrackerAppendBeforeStartNoop(t *testing.T) {
	tr := NewTracker()
	tr.Append("orphan")
	if snap := tr.Snapshot(); len(snap.Blocks) != 0 {
		t.Fatalf("expected no blocks, got %d", len(snap.Blocks))
	}
}

func TestTrackerDuplicateEnd(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("x")
	if _, ok := tr.End(); !ok {
		t.Fatal("first End should succeed")
	}
	if _, ok := tr.End(); ok {
		t.Fatal("duplicate End should report false")
	}
}

func TestTrackerEndWithoutStart(t *testing.T) {
	tr := NewTracker()
	if _, ok := tr.End(); ok {
		t.Fatal("End without Start should report false")
	}
}

func TestTrackerInterrupt(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("partial")
	b, ok := tr.Interrupt()
	if !ok {
		t.Fatal("expected Interrupt to succeed")
	}
	if b.State != BlockInterrupted {
		t.Fatalf("expected BlockInterrupted, got %v", b.State)
	}
}

func TestTrackerReset(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("x")
	tr.End()
	tr.Reset()
	if snap := tr.Snapshot(); len(snap.Blocks) != 0 {
		t.Fatalf("expected empty after Reset, got %d blocks", len(snap.Blocks))
	}
	// Index counter must reset too.
	b := tr.Start()
	if b.Index != 0 {
		t.Fatalf("expected Index reset to 0, got %d", b.Index)
	}
}

func TestTrackerEmptyAppendIgnored(t *testing.T) {
	tr := NewTracker()
	tr.Start()
	tr.Append("")
	b, _ := tr.Active()
	if b.Content != "" {
		t.Fatalf("expected empty content, got %q", b.Content)
	}
}

func TestTrackerConcurrentAccess(t *testing.T) {
	tr := NewTracker()
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tr.Start()
				tr.Append("x")
				tr.End()
				tr.Snapshot()
				tr.Active()
			}
		}()
	}
	wg.Wait()
	// No assertion beyond not racing/panicking; final state must be consistent.
	_ = tr.Snapshot()
}
