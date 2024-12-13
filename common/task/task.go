package task

import (
	"context"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common/signal/semaphore"
)

// OnSuccess executes g() after f() returns nil.
// OnSuccess 在 f() 返回 nil 后执行 g()。
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
// Run 并行执行一系列任务，返回遇到的第一个错误，如果所有任务都通过，则返回 nil。
func Run(ctx context.Context, tasks ...func() error) error {
	fmt.Println("in common-task-task.go func Run")
	n := len(tasks)
	s := semaphore.New(n)
	done := make(chan error, 1)

	for _, task := range tasks {
		<-s.Wait() // 1. 获取一个信号，表示可以执行一个任务
		go func(f func() error) { // 2. 启动一个goroutine执行任务
			err := f() // 3. 执行任务，并将错误保存到err
			if err == nil { // 4. 如果任务执行成功，释放一个信号
				s.Signal()
				return
			}
			// 5. 如果任务执行失败，将错误发送到done通道
			select {
			case done <- err:
			default:
			}
		}(task) // 6. 将当前任务函数 task 作为参数传递给匿名函数 f func()
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
