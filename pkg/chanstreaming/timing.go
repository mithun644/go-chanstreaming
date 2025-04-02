package chanstreaming

import (
	"math/rand"
	"time"
)

// Throttle limits the rate of data emitted from the source channel by aligning the source items with the `interval` ticker events.
func Throttle[T any](interval time.Duration) func(in <-chan T) <-chan T {
	return func(in <-chan T) <-chan T {
		out := make(chan T, 1)
		go func() {
			defer close(out)
			throttle := time.Tick(interval)
			for x := range in {
				<-throttle
				out <- x
			}
		}()
		return out
	}
}

// Jitter introduces random delays to the source channel items.
func Jitter[T any](jitter time.Duration) func(in <-chan T) <-chan T {
	return func(in <-chan T) <-chan T {
		out := make(chan T, 1)
		go func() {
			defer close(out)
			for x := range in {
				time.Sleep(time.Duration(float64(jitter) * (0.5 - rand.Float64())))
				out <- x
			}
		}()
		return out
	}
}
