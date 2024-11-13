package conf

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/proto"

	"github.com/gzjjjfree/gzv2ray-v4/common/protocol"
	"github.com/gzjjjfree/gzv2ray-v4/common/serial"
	"github.com/gzjjjfree/gzv2ray-v4/proxy/mtproto"
)

type MTProtoAccount struct {
	Secret string `json:"secret"`
}

// Build implements Buildable
func (a *MTProtoAccount) Build() (*mtproto.Account, error) {
	if len(a.Secret) != 32 {
		return nil, errors.New("MTProto secret must have 32 chars")
	}
	secret, err := hex.DecodeString(a.Secret)
	if err != nil {
		return nil, errors.New("failed to decode secret: ")
	}
	return &mtproto.Account{
		Secret: secret,
	}, nil
}

type MTProtoServerConfig struct {
	Users []json.RawMessage `json:"users"`
}

func (c *MTProtoServerConfig) Build() (proto.Message, error) {
	config := &mtproto.ServerConfig{}

	if len(c.Users) == 0 {
		return nil, errors.New("zero MTProto users configured")
	}
	config.User = make([]*protocol.User, len(c.Users))
	for idx, rawData := range c.Users {
		user := new(protocol.User)
		if err := json.Unmarshal(rawData, user); err != nil {
			return nil, errors.New("invalid MTProto user")
		}
		account := new(MTProtoAccount)
		if err := json.Unmarshal(rawData, account); err != nil {
			return nil, errors.New("invalid MTProto user")
		}
		accountProto, err := account.Build()
		if err != nil {
			return nil, errors.New("failed to parse MTProto user")
		}
		user.Account = serial.ToTypedMessage(accountProto)
		config.User[idx] = user
	}

	return config, nil
}

type MTProtoClientConfig struct {
}

func (c *MTProtoClientConfig) Build() (proto.Message, error) {
	config := new(mtproto.ClientConfig)
	return config, nil
}
