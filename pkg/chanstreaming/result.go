package chanstreaming

type Result[T any] struct {
	Data  T
	Error error
}

// Catch transforms Result[T] back to just T channel, applying fn function on error to
// give control of error handling
func Catch[T any](fn func(error)) func(in <-chan Result[T]) <-chan T {
	return func(in <-chan Result[T]) <-chan T {
		out := make(chan T, 1)
		go func() {
			defer close(out)
			for x := range in {
				if x.Error != nil {
					fn(x.Error)
					continue
				}
				out <- x.Data
			}
		}()
		return out
	}
}

// Muted transforms the Result[T] back to just T channel, ignoring errors
func Muted[T any](in <-chan Result[T]) <-chan T {
	return Catch[T](func(err error) {})(in)
}

// Panic transforms the Result[T] back to just T channel, panicking on error
func Panic[T any](in <-chan Result[T]) <-chan T {
	return Catch[T](func(err error) { panic(err) })(in)
}

func NewResult[T any](data T) Result[T] {
	return Result[T]{Data: data}
}

func NewError[T any](err error) Result[T] {
	return Result[T]{Error: err}
}

func MapResult[T any, R any](fn func(x T) R) func(in Result[T]) Result[R] {
	return func(in Result[T]) Result[R] {
		if in.Error != nil {
			return NewError[R](in.Error)
		}
		return NewResult(fn(in.Data))
	}
}

func IsError[T any](in Result[T]) bool {
	return in.Error != nil
}

func IsResult[T any](in Result[T]) bool {
	return !IsError(in)
}

func MapError[T any](fn func(error) error) func(in Result[T]) Result[T] {
	return func(in Result[T]) Result[T] {
		if in.Error != nil {
			return NewError[T](in.Error)
		}
		return in
	}
}
