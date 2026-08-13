package reasoning

import (
	"sync"
	"time"
)

// Clock provides the current time. It is injectable so tests can drive
// duration deterministically without sleeping.
type Clock interface {
	Now() time.Time
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

// Option configures a Tracker.
type Option func(*Tracker)

// WithClock overrides the time source used for lifecycle/duration tracking.
func WithClock(c Clock) Option {
	return func(t *Tracker) { t.clock = c }
}

// Tracker is the reasoning lifecycle state machine. It owns the ordered list
// of reasoning blocks, incrementally accumulates deltas into the active block,
// and computes durations. It is safe for concurrent use.
type Tracker struct {
	mu        sync.Mutex
	clock     Clock
	blocks    []Block
	nextIndex int
}

// NewTracker returns a Tracker using the wall clock unless WithClock is given.
func NewTracker(opts ...Option) *Tracker {
	t := &Tracker{clock: wallClock{}}
	for _, o := range opts {
		o(t)
	}
	return t
}

// Start begins a new reasoning block. If a block is already active it is first
// completed implicitly so a new block never pollutes the previous one.
func (t *Tracker) Start() Block {
	t.mu.Lock()
	defer t.mu.Unlock()

	if b := t.activeLocked(); b != nil {
		b.State = BlockCompleted
		b.EndedAt = t.clock.Now()
		b.Duration = b.EndedAt.Sub(b.StartedAt)
	}

	b := Block{
		Index:     t.nextIndex,
		State:     BlockActive,
		StartedAt: t.clock.Now(),
	}
	t.nextIndex++
	t.blocks = append(t.blocks, b)
	return b
}

// Append adds a delta to the active block. It is a no-op when no block is
// active, providing controlled behavior for out-of-order input.
func (t *Tracker) Append(delta string) {
	if delta == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if b := t.activeLocked(); b != nil {
		b.Content += delta
	}
}

// End completes the active block normally. It reports false when no block is
// active (duplicate or orphan End).
func (t *Tracker) End() (Block, bool) {
	return t.finish(BlockCompleted)
}

// Interrupt ends the active block as interrupted (error/abort). It reports
// false when no block is active.
func (t *Tracker) Interrupt() (Block, bool) {
	return t.finish(BlockInterrupted)
}

func (t *Tracker) finish(state BlockState) (Block, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	b := t.activeLocked()
	if b == nil {
		return Block{}, false
	}
	b.State = state
	b.EndedAt = t.clock.Now()
	b.Duration = b.EndedAt.Sub(b.StartedAt)
	return *b, true
}

// Reset clears all blocks and the index counter.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.blocks = nil
	t.nextIndex = 0
}

// Active returns the active block and true if one is in progress.
func (t *Tracker) Active() (Block, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if b := t.activeLocked(); b != nil {
		return *b, true
	}
	return Block{}, false
}

// Snapshot returns an immutable view of all blocks in order.
func (t *Tracker) Snapshot() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	blocks := make([]Block, len(t.blocks))
	copy(blocks, t.blocks)
	return Snapshot{Blocks: blocks}
}

// activeLocked returns the active block or nil. Callers must hold t.mu.
func (t *Tracker) activeLocked() *Block {
	if len(t.blocks) == 0 {
		return nil
	}
	last := &t.blocks[len(t.blocks)-1]
	if last.State == BlockActive {
		return last
	}
	return nil
}
