package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"runtime"
	"os/signal"
	"syscall"

	//"example.com/gztest"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common/cmdarg"
	"github.com/gzjjjfree/gzv2ray-v4/common/platform"
	_ "github.com/gzjjjfree/gzv2ray-v4/main/distro/all"
	_ "github.com/gzjjjfree/gzv2ray-v4/main/loadfile"
)

var (
	configFiles cmdarg.Arg // "Config file for V2Ray.", the option is customed type, parse in main
	configDir   string
	version     = flag.Bool("version", false, "Show current version of GzV2Ray.") //
	test        = flag.Bool("test", false, "Test config file only, without launching GzV2Ray server.")
	format      = flag.String("format", "json", "Format of input file.")

	/* We have to do this here because Golang's Test will also need to parse flag, before
	 * main func in this file is run.
	 */
	_ = func() error { // nolint: unparam  应该会在定义标志时运行
		flag.Var(&configFiles, "config", "Config file for GzV2Ray. Multiple assign is accepted (only json). Latter ones overrides the former ones.")
		flag.Var(&configFiles, "c", "Short alias of -config")
		flag.StringVar(&configDir, "confdir", "", "A dir with multiple json config")

		return nil
	}()
)

func main() {
	
	flag.Parse() //包flag实现命令行flag解析,将命令行解析为定义的标志

	printVersion()

	if *version {
		return
	}

	server, err := startV2Ray()
	if err != nil {
		fmt.Println(err)
		// Configuration error. Exit with a special value to prevent systemd from restarting.
		os.Exit(23)
	}

	if *test {
		fmt.Println("Configuration OK.")
		os.Exit(0)
	}
	fmt.Println("in main-main.go func main server.Start()")
	if err := server.Start(); err != nil {
		fmt.Println("Failed to start", err)
		os.Exit(-1)
	}
	//fmt.Println("in main-main.go func main server.Close()")
	defer server.Close()

	//显式触发 GC 以从配置加载中删除垃圾
	runtime.GC()

	{
		// osSignals: 声明了一个无缓冲的 channel，其类型为 os.Signal，用于接收操作系统信号
		// 创建一个容量为 1 的 channel。容量为 1 表示该 channel 最多只能存储一个信号
		osSignals := make(chan os.Signal, 1)
		// signal.Notify: 这个函数的作用是将指定的信号注册到指定的 channel 上
		// osSignals: 上面创建的 channel，用于接收信号
		// os.Interrupt 和 syscall.SIGTERM: 两个常见的操作系统信号，分别表示用户中断 (Ctrl+C) 和终止进程
		signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
		// <-osSignals: 从 osSignals channel 接收一个信号。当程序执行到这一行时，会阻塞，直到有信号被发送到该 channel
		<-osSignals
	}
}

func printVersion() {
	version := core.VersionStatement()
	for _, s := range version {
		fmt.Println(s)
	}
}

func startV2Ray() (core.Server, error) { //Server 是 V2Ray 的一个实例，任何时候都最多只能有一个 Server 实例在运行。函数返回一个实例
	configFiles := getConfigFilePath()
	//fmt.Println("in startV2ray configFiles: ", configFiles)
	config, err := core.LoadConfig(GetConfigFormat(), configFiles[0], configFiles) // GetConfigFormat() 通过标志 format 定义为 json 文件
	//fmt.Println("in main Print the *config.app:")
	//fmt.Println(config.String())
	//gztest.GetMessageReflectType(config)
	if err != nil {
		return nil, fmt.Errorf("failed to read config files1: [%v]", configFiles.String())
	}
	//fmt.Printf("%+v\n", config)
	server, err := core.New(config)
	if err != nil {
		fmt.Println("113 config is err")
		return nil, fmt.Errorf("failed to create server")
	}

	//fmt.Println("in main.go func startV2Ray() return")

	return server, nil
}

func getConfigFilePath() cmdarg.Arg { // 函数返回字符串数组
	if dirExists(configDir) { // 如何 config 有路径，输出具体路径
		log.Println("Using confdir from arg:", configDir)
		readConfDir(configDir) // 读取路径文件
	} else if envConfDir := platform.GetConfDirPath(); dirExists(envConfDir) { //根据环境变量路径读取 config
		log.Println("Using confdir from env:", envConfDir)
		readConfDir(envConfDir)
	}

	if len(configFiles) > 0 { // 如果已读取到 config.json 文件的路径，返回
		return configFiles
	}

	if workingDir, err := os.Getwd(); err == nil { // Getwd 返回当前目录对应的根路径名。如果当前目录可以通过多条路径到达（由于符号链接），Getwd 可能会返回其中的任意一条。
		// 当没有预设路径时，读取当前目录根路径的 config.json
		configFile := filepath.Join(workingDir, "config.json") // Join 将任意数量的路径元素合并为一个路径，并使用特定于操作系统的 [Separator] 将它们分隔开。空元素将被忽略。
		// 结果为 Cleaned。但是，如果参数列表为空或其所有元素都为空，则 Join 将返回一个空字符串。在 Windows 上，如果第一个非空元素是 UNC 路径，则结果将仅为 UNC 路径
		if fileExists(configFile) {
			log.Println("Using default config: ", configFile)
			return cmdarg.Arg{configFile}
		}
	}

	if configFile := platform.GetConfigurationPath(); fileExists(configFile) { // 按启动当前进程的可执行文件的路径查找 config
		log.Println("Using config from env: ", configFile)
		return cmdarg.Arg{configFile}
	}

	log.Println("Using config from STDIN")
	return cmdarg.Arg{"stdin:"}
}

func dirExists(file string) bool {
	if file == "" {
		return false
	}
	info, err := os.Stat(file)
	return err == nil && info.IsDir()
}

func readConfDir(dirPath string) {
	//confs, err := ioutil.ReadDir(dirPath)
	confs, err := os.ReadDir(dirPath) //ReadDir 读取指定目录，返回按文件名排序的所有目录条目。如果读取目录时发生错误，ReadDir 将返回错误发生前能够读取的条目以及错误。
	if err != nil {
		log.Fatalln(err) // 读取错误输出到日志
	}
	for _, f := range confs {
		if strings.HasSuffix(f.Name(), ".json") { // HasSuffix 报告文件名是否以 .json 结尾
			configFiles.Set(path.Join(dirPath, f.Name())) // Join 将任意数量的路径元素合并为一个路径，并用斜线将它们分隔开。空元素将被忽略。结果为 Cleaned。
			//但是，如果参数列表为空或其所有元素都为空，则 Join 返回一个空字符串。
		}
	}
}

func fileExists(file string) bool {
	info, err := os.Stat(file) // Stat 返回描述指定文件的 [FileInfo]。如果出现错误，则其类型为 [*PathError]。
	return err == nil && !info.IsDir()
}

func GetConfigFormat() string {
	switch strings.ToLower(*format) { // ToLower 返回所有 Unicode 字母都映射为小写字母的 s。
	case "pb", "protobuf":
		return "protobuf"
	default:
		return "json"
	}
}

//func infiniteLoop() {
//    for {
//        fmt.Println("Hello from goroutine!")
 //       time.Sleep(5 * time.Second)
 //   }
//}
