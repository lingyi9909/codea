// Package reasoning normalizes runtime events into a stable Reasoning state
// that the TUI can consume without re-deriving lifecycle, separation, or
// fallback logic. It depends only on the Codea runtime domain package and
// never on the OpenCode vendor layer.
package reasoning

import "time"

// BlockState is the lifecycle state of a single reasoning block.
type BlockState int

const (
	// BlockActive is a reasoning block that is currently receiving deltas.
	BlockActive BlockState = iota
	// BlockCompleted is a reasoning block that ended normally.
	BlockCompleted
	// BlockInterrupted is a reasoning block that ended due to error/abort.
	BlockInterrupted
)

// Block is one contiguous reasoning block. A model answer may contain zero or
// more reasoning blocks, always ordered by Index.
type Block struct {
	// Index is the zero-based position of this block within a session/answer.
	Index int
	// Content is the accumulated reasoning text.
	Content string
	// State is the current lifecycle state of the block.
	State BlockState
	// StartedAt is when the block began.
	StartedAt time.Time
	// EndedAt is when the block ended; zero while Active.
	EndedAt time.Time
	// Duration is EndedAt-StartedAt; zero while Active.
	Duration time.Duration
}

// Snapshot is an immutable view of the tracker's reasoning state.
type Snapshot struct {
	// Blocks lists all reasoning blocks in order, including the active block
	// (if any) as its last element with State == BlockActive.
	Blocks []Block
}

// Active returns the active block, or nil if none is in progress.
func (s Snapshot) Active() *Block {
	if len(s.Blocks) == 0 {
		return nil
	}
	last := &s.Blocks[len(s.Blocks)-1]
	if last.State == BlockActive {
		return last
	}
	return nil
}

// HasActive reports whether a reasoning block is currently in progress.
func (s Snapshot) HasActive() bool {
	return s.Active() != nil
}
