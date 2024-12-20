//go:build !confonly
// +build !confonly

package vmess

import (
	"fmt"
	
	"github.com/gzjjjfree/gzv2ray-v4/common/dice"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol"
	"github.com/gzjjjfree/gzv2ray-v4/common/uuid"
)

// MemoryAccount is an in-memory form of VMess account.
type MemoryAccount struct {
	// ID is the main ID of the account.
	ID *protocol.ID
	// AlterIDs are the alternative IDs of the account.
	AlterIDs []*protocol.ID
	// Security type of the account. Used for client connections.
	Security protocol.SecurityType
}

// AnyValidID returns an ID that is either the main ID or one of the alternative IDs if any.
// AnyValidID 返回一个 ID，该 ID 可以是主 ID，也可以是备用 ID 之一（如果有）。
func (a *MemoryAccount) AnyValidID() *protocol.ID {
	if len(a.AlterIDs) == 0 {
		return a.ID
	}
	return a.AlterIDs[dice.Roll(len(a.AlterIDs))]
}

// Equals implements protocol.Account.
func (a *MemoryAccount) Equals(account protocol.Account) bool {
	fmt.Println("in proxy-vmess-account.go func (a *MemoryAccount) Equals")
	vmessAccount, ok := account.(*MemoryAccount)
	if !ok {
		return false
	}
	// TODO: handle AlterIds difference
	return a.ID.Equals(vmessAccount.ID)
}

// AsAccount implements protocol.Account.
func (a *Account) AsAccount() (protocol.Account, error) {
	id, err := uuid.ParseString(a.Id)
	if err != nil {
		return nil, newError("failed to parse ID").Base(err).AtError()
	}
	protoID := protocol.NewID(id)
	return &MemoryAccount{
		ID:       protoID,
		AlterIDs: protocol.NewAlterIDs(protoID, uint16(a.AlterId)),
		Security: a.SecuritySettings.GetSecurityType(),
	}, nil
}
