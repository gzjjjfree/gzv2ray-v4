package bytespool

import (
	"fmt"
	"sync"
)

func createAllocFunc(size int32) func() any {
	return func() any {
		//fmt.Println("in common-bytespool-pool.go func createAllocFunc(size int32) size: ", size)
		return make([]byte, size)
	}
}

// The following parameters controls the size of buffer pools.
// 以下参数控制缓冲池的大小。
// There are numPools pools. Starting from 2k size, the size of each pool is sizeMulti of the previous one.
// 共有 numPools 个池。从 2k 大小开始，每个池的大小是前一个池的 sizeMulti。
// Package buf is guaranteed to not use buffers larger than the largest pool.
// 包 buf 保证不使用大于最大池的缓冲区。
// Other packets may use larger buffers.
// 其他数据包可能使用更大的缓冲区。
const (
	numPools  = 4
	sizeMulti = 4
)

var (
	pool     [numPools]sync.Pool
	poolSize [numPools]int32
)

func init() {
	fmt.Println("in common-bytespool-pool.go func init")
	size := int32(2048)
	for i := 0; i < numPools; i++ {
		pool[i] = sync.Pool{
			New: createAllocFunc(size),
		}
		//fmt.Println("in common-bytespool-pool.go func ini pool[", i, "], size: ", size)
		poolSize[i] = size
		size *= sizeMulti
	}
}

// GetPool returns a sync.Pool that generates bytes array with at least the given size.
// GetPool 返回一个 sync.Pool，它生成至少具有给定大小的字节数组。
// It may return nil if no such pool exists.
// 如果不存在这样的池，它可能会返回 nil。
// v2ray:api:stable
func GetPool(size int32) *sync.Pool {
	fmt.Println("in common-bytespool-pool.go func GetPool size:", size)
	for idx, ps := range poolSize {
		//fmt.Println("in common-bytespool-pool.go func GetPool idx: %v, ps: %v", idx, ps)
		if size <= ps {
			return &pool[idx]
		}
	}
	return nil
}

// Alloc returns a byte slice with at least the given size. Minimum size of returned slice is 2048.
//
// v2ray:api:stable
func Alloc(size int32) []byte {
	pool := GetPool(size)
	if pool != nil {
		return pool.Get().([]byte)
	}
	return make([]byte, size)
}

// Free puts a byte slice into the internal pool.
// Free 将一个字节切片放入内部池中。
// v2ray:api:stable
func Free(b []byte) {
	size := int32(cap(b))
	b = b[0:cap(b)]
	for i := numPools - 1; i >= 0; i-- {
		if size >= poolSize[i] {
			pool[i].Put(&b) // nolint: staticcheck
			return
		}
	}
}
