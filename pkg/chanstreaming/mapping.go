package chanstreaming

// Mapped applies a transformation function to each element in the source channel without any parallelism.
func Mapped[T any, R any](fn func(T) R) func(in <-chan T) <-chan R {
	return func(in <-chan T) <-chan R {
		out := make(chan R, 1)
		go func() {
			defer close(out)
			for x := range in {
				out <- fn(x)
			}
		}()
		return out
	}
}

// Apply applies a function to each element in the source channel and returns the original element.
// This is useful for side effect with no meaningful results like logging.
func Apply[T any](fn func(T)) func(in <-chan T) <-chan T {
	return Mapped[T, T](func(x T) T {
		fn(x)
		return x
	})
}

// Filter filters the source channel based on the `predicate` function.
func Filter[T any](predicate func(el T) bool) func(in <-chan T) <-chan T {
	return func(in <-chan T) <-chan T {
		out := make(chan T, 1)
		go func() {
			defer close(out)
			for x := range in {
				if predicate(x) {
					out <- x
				}
			}
		}()
		return out
	}
}

// Map applies a transformation function to each element in parallel, preserving order
func Map[T any, R any](fn func(T) R, maxWorkers int) func(in <-chan T) <-chan R {
	return func(in <-chan T) <-chan R {
		// each element of out represents optional 'awaitable' expressed with a size-1 channel.
		// the out itself is a channel limiting the number of concurrent workers
		tasksChannel := make(chan (chan R), maxWorkers)

		// launch input processor
		go func() {
			defer close(tasksChannel)
			for i := range in {
				item := i
				resultChan := make(chan R, 1)
				tasksChannel <- resultChan
				go func() {
					result := fn(item)
					resultChan <- result
					close(resultChan)
				}()
			}
		}()

		// launch output consumer
		outChannel := make(chan R, 1)
		go func() {
			defer close(outChannel)
			for resultChan := range tasksChannel {
				result := <-resultChan
				outChannel <- result
			}
		}()

		return outChannel
	}
}

// MapUnordered applies a transformation function in parallel without preserving order
func MapUnordered[T any, R any](fn func(T) R, maxWorkers int) func(in <-chan T) <-chan R {
	return func(in <-chan T) <-chan R {
		out := make(chan R, maxWorkers)
		// the pure form of sync.WaitGroup
		doneQueue := make(chan struct{}, maxWorkers)

		// launch maxWorkers
		for i := 0; i < maxWorkers; i++ {
			go func() {
				for item := range in {
					result := fn(item)
					out <- result
				}
				doneQueue <- struct{}{}
			}()
		}

		go func() {
			// await all maxWorkers
			for i := 0; i < maxWorkers; i++ {
				<-doneQueue
			}
			close(doneQueue)
			// close the output channel
			close(out)
		}()
		return out
	}
}
