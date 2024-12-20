package filesystem

import (
	"io"
	"os"
	"fmt"
	//"errors"

	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/platform"
)

type FileReaderFunc func(path string) (io.ReadCloser, error)

type FileWriterFunc func(path string) (io.WriteCloser, error)

var NewFileReader FileReaderFunc = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

var NewFileWriter FileWriterFunc = func(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

func ReadFile(path string) ([]byte, error) {
	fmt.Println("in common-platform-filesystem-file.go func ReadFile: ", path)
	// 将路径path 的文件读入reader 中
	reader, err := NewFileReader(path)
	if err != nil {
		fmt.Println("in common-platform-filesystem-file.go func ReadFile err != nil: ", err)
		return nil, err
	}
	defer reader.Close()
	//fmt.Println("in common-platform-filesystem-file.go func ReadFile: is err")
	//return nil, errors.New("is err")
	return buf.ReadAllToBytes(reader)
}

func WriteFile(path string, payload []byte) error {
	fmt.Println("in common-platform-filesystem-file.go func WriteFile")
	writer, err := NewFileWriter(path)
	if err != nil {
		return err
	}
	defer writer.Close()

	return buf.WriteAllBytes(writer, payload)
}

func ReadAsset(file string) ([]byte, error) {
	return ReadFile(platform.GetAssetLocation(file))
}

func CopyFile(dst string, src string) error {
	fmt.Println("in common-platform-filesystem-file.go func CopyFile")
	bytes, err := ReadFile(src)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(bytes)
	return err
}
