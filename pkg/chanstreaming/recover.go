package chanstreaming

import (
	"errors"
	"fmt"
)

func NewAsyncResult[T any](fn func() T) <-chan Result[T] {
	output := make(chan Result[T], 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// check if the panic is an error
				if _, ok := r.(error); ok {
					result := Result[T]{
						Error: r.(error),
					}
					output <- result
				} else {
					// convert panic to error message str
					result := Result[T]{
						Error: errors.New(fmt.Sprint(r)),
					}
					output <- result
				}
			}
			close(output)
		}()
		output <- Result[T]{
			Data: fn(),
		}
	}()
	return output
}

func MapSafeAsync[T any, R any](fn func(T) R, maxWorkers int) func(in <-chan T) <-chan <-chan Result[R] {
	return Map[T, <-chan Result[R]](func(in T) <-chan Result[R] {
		return NewAsyncResult[R](func() R { return fn(in) })
	}, maxWorkers)
}

func MapSafe[T any, R any](fn func(T) R, maxWorkers int) func(in <-chan T) <-chan Result[R] {
	return func(in <-chan T) <-chan Result[R] {
		x := MapSafeAsync[T, R](fn, maxWorkers)(in)
		flattened := FlatMap[<-chan Result[R], Result[R]](func(asyncResult <-chan Result[R]) <-chan Result[R] {
			return asyncResult
		})(x)
		return flattened
	}
}

func MapUnorderedSafeAsync[T any, R any](fn func(T) R, maxWorkers int) func(in <-chan T) <-chan <-chan Result[R] {
	return MapUnordered[T, <-chan Result[R]](func(in T) <-chan Result[R] {
		return NewAsyncResult[R](func() R { return fn(in) })
	}, maxWorkers)
}

func MapUnorderedSafe[T any, R any](fn func(T) R, maxWorkers int) func(in <-chan T) <-chan Result[R] {
	return func(in <-chan T) <-chan Result[R] {
		x := MapUnorderedSafeAsync[T, R](fn, maxWorkers)(in)
		flattened := FlatMap[<-chan Result[R], Result[R]](func(asyncResult <-chan Result[R]) <-chan Result[R] {
			return asyncResult
		})(x)
		return flattened
	}
}
