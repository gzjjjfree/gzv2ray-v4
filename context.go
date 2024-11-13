// +build !confonly

package core

import (
	"context"
	"fmt"
)

/* toContext returns ctx from the given context, or creates an Instance if the context doesn't find that.

It is unsupported to use this function to create a context that is suitable to invoke V2Ray's internal component
in third party code, you shouldn't use //go:linkname to alias of this function into your own package and
use this function in your third party code.

For third party code, usage enabled by creating a context to interact with V2Ray's internal component is unsupported,
and may break at any time.

toContext 从给定的上下文返回 ctx，如果上下文找不到，则创建一个实例
不支持使用该函数在第三方代码中创建适合调用 V2Ray 内部组件的上下文，您不应使用 //go:linkname 将该函数的别名放入您自己的包中并在第三方代码中使用该函数。
对于第三方代码，通过创建上下文与 V2Ray 内部组件交互而实现的使用不受支持，并且可能随时中断。
*/
func toContext(ctx context.Context, v *Instance) context.Context {
//fmt.Println("in context.go func toContext")
	if FromContext(ctx) != v {
		//fmt.Println("in context.go func toContext FromContext(ctx) != v")
		ctx = context.WithValue(ctx, gzv2rayKey, v)
		if v.featureResolutions == nil {
			fmt.Println("in context.go func toContext v.featureResolutions == nil")
		} else {
			fmt.Println("in context.go func toContext v.featureResolutions != nil")
		}
	}
	return ctx
}

// FromContext returns an Instance from the given context, or nil if the context doesn't contain one.
// FromContext 从给定的上下文返回一个实例，如果上下文不包含实例，则返回 nil
func FromContext(ctx context.Context) *Instance {
	//fmt.Println("in context.go func FromContext")
	if s, ok := ctx.Value(gzv2rayKey).(*Instance); ok {
		//fmt.Println("in context.go func FromContext ok")
		return s
	}
	return nil
}

// V2rayKey is the key type of Instance in Context, exported for test.
// V2rayKey是Context中Instance的密钥类型，用于测试导出。
type gzV2rayKey int

const gzv2rayKey gzV2rayKey = 1

// MustFromContext returns an Instance from the given context, or panics if not present.
// MustFromContext 从给定的上下文返回一个实例，如果不存在则会引起恐慌
func MustFromContext(ctx context.Context) *Instance {
	v := FromContext(ctx)
	if v == nil {
		panic("V is not in context.")
	}
	return v
}

/* mustToContext returns ctx from the given context, or panics if not found that.

It is unsupported to use this function to create a context that is suitable to invoke V2Ray's internal component
in third party code, you shouldn't use //go:linkname to alias of this function into your own package and
use this function in your third party code.

For third party code, usage enabled by creating a context to interact with V2Ray's internal component is unsupported,
and may break at any time.

*/
//func mustToContext(ctx context.Context, v *Instance) context.Context {
//	if c := toContext(ctx, v); c != ctx {
//		panic("V is not in context.")
//	}
//	return ctx
//}
