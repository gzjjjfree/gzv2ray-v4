package loadfile

import (
	"embed"
	"fmt"

	//"io"
	"os"

	"github.com/gzjjjfree/gzv2ray-v4/common/platform"
)

//go:embed geosite.dat
var f embed.FS

//go:embed geoip.dat
var d embed.FS



func Loadfile(filename string, f embed.FS) error {

	data, err := f.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return err
	}
	// fmt.Println(string(data))

	// 要写入的字符串
	// str := "Hello, world!\nThis is a test string."
	path := platform.GetAssetLocation(filename)
	// 创建文件
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer file.Close() // 延迟关闭文件

	// 写入字符串
	_, err = file.Write(data)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	fmt.Println("geosite.dat written to file successfully!")
	return nil
}

func init() {

	fmt.Println("run main-loadfile func init: ")
	Loadfile("geosite.dat", f)
	Loadfile("geoip.dat", d)
}
