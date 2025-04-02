package chanstreaming

// Buffered returns a channel that is backed by another channel with the given size
func Buffered[T any](size int) func(in <-chan T) <-chan T {
	return func(in <-chan T) <-chan T {
		out := make(chan T, size)
		go func() {
			defer close(out)
			for x := range in {
				out <- x
			}
		}()
		return out
	}
}
