package protocol

import (
	"errors"
)

func (u *User) GetTypedAccount() (Account, error) {
	if u.GetAccount() == nil {
		return nil, errors.New("Account missing")
	}

	rawAccount, err := u.Account.GetInstance()
	if err != nil {
		return nil, err
	}
	if asAccount, ok := rawAccount.(AsAccount); ok {
		return asAccount.AsAccount()
	}
	if account, ok := rawAccount.(Account); ok {
		return account, nil
	}
	return nil, errors.New("Unknown account type: ")
}

func (u *User) ToMemoryUser() (*MemoryUser, error) {
	account, err := u.GetTypedAccount()
	if err != nil {
		return nil, err
	}
	return &MemoryUser{
		Account: account,
		Email:   u.Email,
		Level:   u.Level,
	}, nil
}

// MemoryUser is a parsed form of User, to reduce number of parsing of Account proto.
// MemoryUser 是 User 的解析形式，以减少 Account 协议的解析次数
type MemoryUser struct {
	// Account is the parsed account of the protocol.
	Account Account
	Email   string
	Level   uint32
}
