package chanstreaming

import (
	"errors"
	"time"
)

// Scan produces a channel that takes initialState then evolves the state with each new element from consuming from the original source.
// The resulting channel will emit a new state on each new source element.
func Scan[TIn any, TState any](fn func(TState, TIn) TState, initialState TState) func(in <-chan TIn) <-chan TState {
	return func(in <-chan TIn) <-chan TState {
		out := make(chan TState, 1)
		go func() {
			defer close(out)
			state := initialState
			for x := range in {
				state = fn(state, x)
				out <- state
			}
		}()
		return out
	}
}

// Fold works similarly to Scan, difference that only the last value of the accumulated state is sent to the resulting channel.
func Fold[TIn any, TState any](fn func(TState, TIn) TState, initialState TState) func(in <-chan TIn) <-chan TState {
	return func(in <-chan TIn) <-chan TState {
		out := make(chan TState, 1)
		go func() {
			defer close(out)
			state := initialState
			for x := range in {
				state = fn(state, x)
			}
			out <- state
		}()
		return out
	}
}

type timedWindowElement[T any] struct {
	data T
	ts   time.Time
}

type OverflowStrategy int

const (
	Ignore   OverflowStrategy = iota // this will attempt to add the element to the window, risking running out of memory
	Error                            // this will panic if the window overflows
	DropTail                         // this will ignore the new element of the window until the oldest elements are dropped due to expiration
	DropHead                         // this will drop the oldest elements in the window to make room for the new element
)

type WindowConfig struct {
	MaxSize          int
	Duration         time.Duration
	OverflowStrategy OverflowStrategy
}

var unbound = WindowConfig{}

type WindowOverflowError error

func NewWindowOverflowError() WindowOverflowError {
	return WindowOverflowError(errors.New("window overflow"))
}

// WithSlidingWindow creates a sliding window of elements with behavior as specified in the windowConfig.
func WithSlidingWindow[T any](windowConfig WindowConfig) func(in <-chan T) <-chan []T {
	if windowConfig == unbound {
		panic("window config must be set")
	}

	zer := []timedWindowElement[T]{}
	scanner := Scan[T, []timedWindowElement[T]](func(state []timedWindowElement[T], x T) []timedWindowElement[T] {
		tsNow := time.Now()
		if windowConfig.Duration != 0 {
			dropCount := 0
			for i := 0; i < len(state); i++ {
				duration := tsNow.Sub(state[i].ts)
				if duration > windowConfig.Duration {
					dropCount++
				} else {
					break
				}
			}
			if len(state) > 0 && dropCount > 0 {
				state = state[dropCount:]
			}
		}

		if windowConfig.OverflowStrategy == Ignore || windowConfig.MaxSize == 0 {
			state = append(state, timedWindowElement[T]{
				data: x,
				ts:   tsNow,
			})
		} else {
			if len(state) < windowConfig.MaxSize {
				state = append(state, timedWindowElement[T]{
					data: x,
					ts:   tsNow,
				})
			} else {
				switch windowConfig.OverflowStrategy {
				case DropHead:
					state = state[1:]
					state = append(state, timedWindowElement[T]{
						data: x,
						ts:   tsNow,
					})
					break
				case DropTail:
					break
				case Error:
				default:
					panic(NewWindowOverflowError())
				}
			}
		}

		if len(state) > 0 {
			stateCopy := make([]timedWindowElement[T], len(state))
			copy(stateCopy, state)
			return stateCopy
		} else {
			return state
		}
	}, zer)

	batchMapper := func(x []timedWindowElement[T]) []T {
		dataSlice := make([]T, len(x))
		for i, v := range x {
			dataSlice[i] = v.data
		}
		return dataSlice
	}

	mapper := Mapped[[]timedWindowElement[T], []T](batchMapper)

	return func(in <-chan T) <-chan []T {
		scanned := scanner(in)
		mapped := mapper(scanned)
		return mapped
	}
}

func (c *WindowConfig) WithMaxSize(maxSize int) WindowConfig {
	return WindowConfig{
		OverflowStrategy: DropHead,
		MaxSize:          maxSize,
		Duration:         c.Duration,
	}
}

func (c *WindowConfig) WithDuration(duration time.Duration) WindowConfig {
	// produce the config copy with new duration
	return WindowConfig{
		Duration:         duration,
		MaxSize:          c.MaxSize,
		OverflowStrategy: c.OverflowStrategy,
	}
}

func (c *WindowConfig) WithOverflowStrategy(overflowStrategy OverflowStrategy) WindowConfig {
	// produce the config copy with new overflow strategy
	return WindowConfig{
		MaxSize:          c.MaxSize,
		Duration:         c.Duration,
		OverflowStrategy: overflowStrategy,
	}
}

// WithSlidingWindowTimed creates a sliding window of elements bound by duration.
// The window is advanced by the time elapsed since the first element was added to the window.
func WithSlidingWindowTimed[T any](windowDuration time.Duration) func(in <-chan T) <-chan []T {
	theConfig := unbound
	theConfig = theConfig.WithDuration(windowDuration)
	result := WithSlidingWindow[T](theConfig)
	return result
}

// WithSlidingWindowCount creates a sliding window of elements bound by window length.
// The window is advanced by dropping the oldest element from it.
func WithSlidingWindowCount[T any](windowSize int) func(in <-chan T) <-chan []T {
	theConfig := unbound
	theConfig = theConfig.WithMaxSize(windowSize)
	theConfig = theConfig.WithOverflowStrategy(DropHead)
	result := WithSlidingWindow[T](theConfig)
	return result
}
