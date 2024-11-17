package platform

import (
	"os"
	"strings"
	"strconv"
	"path/filepath"
)

// GetConfDirPath reads "gzv2ray.location.confdir"
func GetConfDirPath() string {
	const name = "gzv2ray.location.confdir"
	configPath := NewEnvFlag(name).GetValue(func() string { return "" })
	return configPath
}

type EnvFlag struct {
	Name    string
	AltName string
}

func NewEnvFlag(name string) EnvFlag { //字符组
	return EnvFlag{
		Name:    name,                   // 输入的字符
		AltName: NormalizeEnvName(name), // 处理后的字符
	}
}

func NormalizeEnvName(name string) string { // 处理路径字符 ReplaceAll 替换字符，ToUpper 转大写，TrimSpace 去除前后空格
	return strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(name)), ".", "_")
}

func (f EnvFlag) GetValue(defaultValue func() string) string { //EnvFlag 类型的方法 GetValue 参数为返回 string 的函数，方法返回 string
	if v, found := os.LookupEnv(f.Name); found { // os.LookupEnv 检索由键命名的环境变量的值。如果变量存在于环境中，则返回值（可能为空）且布尔值为真。否则返回值将为空且布尔值为假。
		return v // 有 v2ray.location.confdir 的环境变量，返回变量的值
	}
	if len(f.AltName) > 0 {
		if v, found := os.LookupEnv(f.AltName); found {
			return v
		}
	}
	return defaultValue() // defaultValue 默认返回函数，返回空值
}

func (f EnvFlag) GetValueAsInt(defaultValue int) int {
	useDefaultValue := false
	s := f.GetValue(func() string {
		useDefaultValue = true
		return ""
	})
	if useDefaultValue {
		return defaultValue
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int(v)
}

func getExecutableDir() string {
	exec, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exec)
}

func getExecutableSubDir(dir string) func() string {
	return func() string {
		return filepath.Join(getExecutableDir(), dir)
	}
}

func GetPluginDirectory() string {
	const name = "gzv2ray.location.plugin"
	pluginDir := NewEnvFlag(name).GetValue(getExecutableSubDir("plugins"))
	return pluginDir
}

func GetConfigurationPath() string {
	const name = "gzv2ray.location.config"
	configPath := NewEnvFlag(name).GetValue(getExecutableDir)
	return filepath.Join(configPath, "config.json")
}

