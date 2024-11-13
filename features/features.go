package features

import (
	"fmt"

    "github.com/gzjjjfree/gzv2ray-v4/common"
)



// Feature is the interface for V2Ray features. All features must implement this interface.
// Feature 是 V2Ray 特性的接口，所有特性都必须实现此接口。
// All existing features have an implementation in app directory. These features can be replaced by third-party ones.
// 所有现有功能在 app 目录中均有实现。这些功能可以被第三方功能替换。
type Feature interface {
	common.HasType
	common.Runnable
}

// PrintDeprecatedFeatureWarning prints a warning for deprecated feature.
// PrintDeprecatedFeatureWarning 打印有关已弃用功能的警告。
func PrintDeprecatedFeatureWarning(feature string) {
	fmt.Println("You are using a deprecated feature: . Please update your config file with latest configuration format, or update your client software.")
}