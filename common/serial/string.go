package serial

import (
	"fmt"
	"strings"
)

// ToString serialize an arbitrary value into string.
// ToString 将任意值序列化为字符串。
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch value := v.(type) {
	case string:
		return value
	case *string:
		return *value
	case fmt.Stringer:
		return value.String()
	case error:
		return value.Error()
	default:
		return fmt.Sprintf("%+v", value)
	}
}

// Concat concatenates all input into a single string.
// Concat 将所有输入连接成一个字符串。
func Concat(v ...interface{}) string {
	builder := strings.Builder{}
	for _, value := range v {
		builder.WriteString(ToString(value))
	}
	return builder.String()
}
