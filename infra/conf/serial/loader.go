package serial

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"example.com/gztest"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/infra/conf"
	json_reader "github.com/gzjjjfree/gzv2ray-v4/infra/conf/json"
)

type offset struct {
	line int
	char int
}

func findOffset(b []byte, o int) *offset {
	if o >= len(b) || o < 0 {
		return nil
	}

	line := 1
	char := 0
	for i, x := range b {
		if i == o {
			break
		}
		if x == '\n' {
			line++
			char = 0
		} else {
			char++
		}
	}

	return &offset{line: line, char: char}
}

// DecodeJSONConfig reads from reader and decode the config into *conf.Config
// syntax error could be detected.
//func DecodeJSONConfig(reader io.Reader) (*conf.Config, error) {
//	jsonConfig := &conf.Config{}

//	jsonContent := bytes.NewBuffer(make([]byte, 0, 10240))
//	jsonReader := io.TeeReader(&json_reader.Reader{
//		Reader: reader,
//	}, jsonContent)
//	decoder := json.NewDecoder(jsonReader)

//		if err := decoder.Decode(jsonConfig); err != nil {
//			var pos *offset
//			cause := errors.Unwrap(err)
//			switch tErr := cause.(type) {
//			case *json.SyntaxError:
//				pos = findOffset(jsonContent.Bytes(), int(tErr.Offset))
//			case *json.UnmarshalTypeError:
//				pos = findOffset(jsonContent.Bytes(), int(tErr.Offset))
//			}
//			if pos != nil {
//				return nil, errors.New("failed to read config file at line")
//			}
//			return nil, errors.New("failed to read config file")
//		}
//	 return jsonConfig, nil
//	}
//
// DecodeJSONConfig reads from reader and decode the config into *conf.Config
// syntax error could be detected.
func DecodeJSONConfig(reader io.Reader) (*conf.Config, error) {
	jsonConfig := &conf.Config{}
	err := DecodeJSON(reader, jsonConfig)
	if err != nil {
		return nil, err
	}
	//fmt.Println("in infra-conf-serial-loader.go func DecodeJSONConffig the jsonConfig is:")
	//gztest.GetMessageReflectType(*jsonConfig)
	return jsonConfig, nil
}

// DecodeJSON reads from reader and decode into target
// syntax error could be detected.
func DecodeJSON(reader io.Reader, target interface{}) error {
	fmt.Println("in infra-conf-serial-loader.go func DecodeJSON ")
	jsonContent := bytes.NewBuffer(make([]byte, 0, 10240))
	jsonReader := io.TeeReader(&json_reader.Reader{
		Reader: reader,
	}, jsonContent)
	decoder := json.NewDecoder(jsonReader)
	//fmt.Println("in infra-conf-serial-loader.go func DecodeJSON decoder is:")
	///gztest.GetMessageReflectType(decoder.Buffered)
	if err := decoder.Decode(target); err != nil {
		fmt.Println("in infra-conf-serial-loader.go func DecodeJSON err != nil:")
		var pos *offset
		cause := gztest.Cause(err)
		switch tErr := cause.(type) {
		case *json.SyntaxError:
			pos = findOffset(jsonContent.Bytes(), int(tErr.Offset))
		case *json.UnmarshalTypeError:
			pos = findOffset(jsonContent.Bytes(), int(tErr.Offset))
		}
		if pos != nil {
			return newError("failed to read config file at line ", pos.line, " char ", pos.char).Base(err)
		}
		return newError("failed to read config file").Base(err)
	}

	return nil
}

func LoadJSONConfig(reader io.Reader) (*core.Config, error) {
	jsonConfig, err := DecodeJSONConfig(reader)
	if err != nil {
		return nil, err
	}

	pbConfig, err := jsonConfig.Build()
	if err != nil {
		return nil, errors.New("failed to parse json config")
	}

	return pbConfig, nil
}
