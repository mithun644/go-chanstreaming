package chanstreaming

import (
	"errors"
	"fmt"
)

// UnfoldSafe creates a channel that unfolds the state into data elements asynchronously.
// returns a tuple of State, Data, Continued
// state0 -> state1, data0, true => continue unfolding, emit data0
// state1 -> state2, data1, true => continue unfolding, emit data1
// state2 -> _, _, false => stop unfolding
func UnfoldSafe[TState any, TData any](fn func(s TState) (TState, TData, bool), zero TState) <-chan Result[TData] {
	output := make(chan Result[TData], 1)
	// launch worker goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// check if the panic is an error
				if _, ok := r.(error); ok {
					output <- Result[TData]{Error: r.(error)}
				} else {
					// convert panic to error message str
					output <- Result[TData]{Error: errors.New(fmt.Sprint(r))}
				}
			}
			close(output)
		}()

		st := zero
		for {
			newSt, data, continued := fn(st)
			if !continued {
				break
			} else {
				output <- Result[TData]{Data: data}
				st = newSt
			}
		}
	}()
	return output
}
