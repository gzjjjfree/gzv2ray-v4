package common

import (
	"fmt"
	"errors"
)



// Runnable is the interface for objects that can start to work and stop on demand.
type Runnable interface { // Runnable 是可以根据需要开始工作和停止的对象的接口。
	// Start starts the runnable object. Upon the method returning nil, the object begins to function properly.
	//Start 启动可运行对象。当该方法返回 nil 时，该对象开始正常运行
	Start() error

	Closable
}

// Closable is the interface for objects that can release its resources.
//
// v2ray:api:beta
type Closable interface {
	// Close release all resources used by this object, including goroutines.
	//Close 释放此对象使用的所有资源，包括 goroutines
	Close() error
}

// HasType is the interface for objects that knows its type.
type HasType interface { // HasType 是知道其类型的对象的接口
	// Type returns the type of the object.
	// Usually it returns (*Type)(nil) of the object.
	Type() interface{}
}

// Close closes the obj if it is a Closable.
// 如果 obj 是可关闭的，则 Close 将关闭它。
// v2ray:api:beta
func Close(obj interface{}) error {
	fmt.Println("in common-interfaces.go func Close(obj interface{})")
	if c, ok := obj.(Closable); ok {
		fmt.Println("in common-interfaces.go func Close(obj interface{}) return c.Close()")
		return c.Close()
	}
	return nil
}

// Interrupt calls Interrupt() if object implements Interruptible interface, or Close() if the object implements Closable interface.
// 如果对象实现了 Interruptible 接口，则 Interrupt 调用 Interrupt()；如果对象实现了 Closable 接口，则 Interrupt 调用 Close()。
//
// v2ray:api:beta
func Interrupt(obj interface{}) error {
	if c, ok := obj.(Interruptible); ok {
		c.Interrupt()
		return nil
	}
	return Close(obj)
}

// Interruptible is an interface for objects that can be stopped before its completion.
//
// v2ray:api:beta
type Interruptible interface {
	Interrupt()
}

// ChainedClosable is a Closable that consists of multiple Closable objects.
type ChainedClosable []Closable

// Close implements Closable.
func (cc ChainedClosable) Close() error {
	fmt.Println("in common-interfaces.go func  (cc ChainedClosable) Close()")
	var errs []error
	for _, c := range cc {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

