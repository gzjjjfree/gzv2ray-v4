package task

import "github.com/gzjjjfree/gzv2ray-v4/common"

// Close returns a func() that closes v.
func Close(v interface{}) func() error {
	return func() error {
		return common.Close(v)
	}
}
