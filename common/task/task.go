package task

import (
	"context"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common/signal/semaphore"
)

// OnSuccess executes g() after f() returns nil.
func OnSuccess(f func() error, g func() error) func() error {
	fmt.Println("in common-task-task.go func OnSuccess")
	return func() error {
		if err := f(); err != nil {
			return err
		}
		return g()
	}
}

// Run executes a list of tasks in parallel, returns the first error encountered or nil if all tasks pass.
func Run(ctx context.Context, tasks ...func() error) error {
	fmt.Println("in common-task-task.go func Run")
	n := len(tasks)
	s := semaphore.New(n)
	done := make(chan error, 1)

	for _, task := range tasks {
		<-s.Wait()
		go func(f func() error) {
			err := f()
			if err == nil {
				s.Signal()
				return
			}

			select {
			case done <- err:
			default:
			}
		}(task)
	}

	for i := 0; i < n; i++ {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-s.Wait():
		}
	}

	return nil
}
