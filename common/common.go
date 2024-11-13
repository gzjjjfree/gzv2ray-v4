package common

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

)

var (
	// ErrNoClue is for the situation that existing information is not enough to make a decision. For example, Router may return this error when there is no suitable route.
	ErrNoClue = errors.New("not enough information for making a decision")
)

// Must panics if err is not nil.
// 如果 err 不为零，则必须恐慌。
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Must2 panics if the second parameter is not nil, otherwise returns the first parameter.
// 如果第二个参数不为零，则 Must2 会发生恐慌，否则返回第一个参数。
func Must2(v interface{}, err error) interface{} {
	Must(err)
	return v
}


// Error2 returns the err from the 2nd parameter.
// Error2 返回第二个参数的错误。
func Error2(v interface{}, err error) error {
	return err
}

// envFile returns the name of the Go environment configuration file.
// Copy from https://github.com/golang/go/blob/c4f2a9788a7be04daf931ac54382fbe2cb754938/src/cmd/go/internal/cfg/cfg.go#L150-L166
func envFile() (string, error) {
	if file := os.Getenv("GOENV"); file != "" {
		if file == "off" {
			return "", fmt.Errorf("GOENV=off")
		}
		return file, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("missing user-config dir")
	}
	return filepath.Join(dir, "go", "env"), nil
}

// GetRuntimeEnv returns the value of runtime environment variable,
// that is set by running following command: `go env -w key=value`.
func GetRuntimeEnv(key string) (string, error) {
	file, err := envFile()
	if err != nil {
		return "", err
	}
	if file == "" {
		return "", fmt.Errorf("missing runtime env file")
	}
	var data []byte
	var runtimeEnv string
	data, readErr := os.ReadFile(file)
	if readErr != nil {
		return "", readErr
	}
	envStrings := strings.Split(string(data), "\n")
	for _, envItem := range envStrings {
		envItem = strings.TrimSuffix(envItem, "\r")
		envKeyValue := strings.Split(envItem, "=")
		if strings.EqualFold(strings.TrimSpace(envKeyValue[0]), key) {
			runtimeEnv = strings.TrimSpace(envKeyValue[1])
		}
	}
	return runtimeEnv, nil
}

// GetGOBIN returns GOBIN environment variable as a string. It will NOT be empty.
func GetGOBIN() string {
	// The one set by user explicitly by `export GOBIN=/path` or `env GOBIN=/path command`
	GOBIN := os.Getenv("GOBIN")
	if GOBIN == "" {
		var err error
		// The one set by user by running `go env -w GOBIN=/path`
		GOBIN, err = GetRuntimeEnv("GOBIN")
		if err != nil {
			// The default one that Golang uses
			return filepath.Join(build.Default.GOPATH, "bin")
		}
		if GOBIN == "" {
			return filepath.Join(build.Default.GOPATH, "bin")
		}
		return GOBIN
	}
	return GOBIN
}

// GetGOPATH returns GOPATH environment variable as a string. It will NOT be empty.
func GetGOPATH() string {
	// The one set by user explicitly by `export GOPATH=/path` or `env GOPATH=/path command`
	GOPATH := os.Getenv("GOPATH")
	if GOPATH == "" {
		var err error
		// The one set by user by running `go env -w GOPATH=/path`
		GOPATH, err = GetRuntimeEnv("GOPATH")
		if err != nil {
			// The default one that Golang uses
			return build.Default.GOPATH
		}
		if GOPATH == "" {
			return build.Default.GOPATH
		}
		return GOPATH
	}
	return GOPATH
}

// FetchHTTPContent dials http(s) for remote content
func FetchHTTPContent(target string) ([]byte, error) {
	parsedTarget, err := url.Parse(target)
	if err != nil {
		return nil, errors.New("invalid URL: ")
	}

	if s := strings.ToLower(parsedTarget.Scheme); s != "http" && s != "https" {
		return nil, errors.New("invalid scheme: ")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL:    parsedTarget,
		Close:  true,
	})
	if err != nil {
		return nil, errors.New("failed to dial to ")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected HTTP status code: ")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read HTTP response")
	}

	return content, nil
}
