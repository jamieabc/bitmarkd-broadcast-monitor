package tasks

type executionFunction func([]interface{})

// Tasks - interface for Task to run
// Expect to receive signal from done channel when every goroutine finishes
// closing procedure
type Tasks interface {
	Go(executionFunction, ...interface{})
	Done()
}
