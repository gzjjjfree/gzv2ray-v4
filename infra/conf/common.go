package conf

import (
	"encoding/json"
	"os"
	"strings"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol"
)

type StringList []string

func NewStringList(raw []string) *StringList {
	fmt.Println("in infra-conf-common.go func  NewStringList")
	list := StringList(raw)
	return &list
}

func (v StringList) Len() int {
	return len(v)
}

func (v *StringList) UnmarshalJSON(data []byte) error {
	fmt.Println("in infra-conf-common.go func (v *StringList) UnmarshalJSON")
	var strarray []string
	if err := json.Unmarshal(data, &strarray); err == nil {
		*v = *NewStringList(strarray)
		return nil
	}

	var rawstr string
	if err := json.Unmarshal(data, &rawstr); err == nil {
		strlist := strings.Split(rawstr, ",")
		*v = *NewStringList(strlist)
		return nil
	}
	return errors.New("unknown format of a string list")
}

type Address struct {
	net.Address
}

func (v *Address) UnmarshalJSON(data []byte) error {
	fmt.Println("in infra-conf-common.go func  (v *Address) UnmarshalJSON")
	var rawStr string
	if err := json.Unmarshal(data, &rawStr); err != nil {
		return errors.New("invalid address")
	}
	v.Address = net.ParseAddress(rawStr)

	return nil
}

func (v *Address) Build() *net.IPOrDomain {
	fmt.Println("in infra-conf-common.go func (v *Address) Build()")
	return net.NewIPOrDomain(v.Address)
}

type Network string

func (v Network) Build() net.Network {
	fmt.Println("in infra-conf-common.go func (v Network) Build()")
	switch strings.ToLower(string(v)) {
	case "tcp":
		return net.Network_TCP
	case "udp":
		return net.Network_UDP
	case "unix":
		return net.Network_UNIX
	default:
		return net.Network_Unknown
	}
}

type NetworkList []Network

func (v *NetworkList) UnmarshalJSON(data []byte) error {
	fmt.Println("in infra-conf-common.go func  (v *NetworkList) UnmarshalJSON")
	var strarray []Network
	if err := json.Unmarshal(data, &strarray); err == nil {
		nl := NetworkList(strarray)
		*v = nl
		return nil
	}

	var rawstr Network
	if err := json.Unmarshal(data, &rawstr); err == nil {
		strlist := strings.Split(string(rawstr), ",")
		nl := make([]Network, len(strlist))
		for idx, network := range strlist {
			nl[idx] = Network(network)
		}
		*v = nl
		return nil
	}
	return errors.New("unknown format of a string list")
}

func (v *NetworkList) Build() []net.Network {
	fmt.Println("in infra-conf-common.go func (v *NetworkList) Build()")
	if v == nil {
		return []net.Network{net.Network_TCP}
	}

	list := make([]net.Network, 0, len(*v))
	for _, network := range *v {
		list = append(list, network.Build())
	}
	return list
}

func parseIntPort(data []byte) (net.Port, error) {
	fmt.Println("in infra-conf-common.go func parseIntPort")
	var intPort uint32
	err := json.Unmarshal(data, &intPort)
	if err != nil {
		return net.Port(0), err
	}
	return net.PortFromInt(intPort)
}

func parseStringPort(s string) (net.Port, net.Port, error) {
	fmt.Println("in infra-conf-common.go func  parseStringPort")
	if strings.HasPrefix(s, "env:") {
		s = s[4:]
		s = os.Getenv(s)
	}

	pair := strings.SplitN(s, "-", 2)
	if len(pair) == 0 {
		return net.Port(0), net.Port(0), errors.New("invalid port range")
	}
	if len(pair) == 1 {
		port, err := net.PortFromString(pair[0])
		return port, port, err
	}

	fromPort, err := net.PortFromString(pair[0])
	if err != nil {
		return net.Port(0), net.Port(0), err
	}
	toPort, err := net.PortFromString(pair[1])
	if err != nil {
		return net.Port(0), net.Port(0), err
	}
	return fromPort, toPort, nil
}

func parseJSONStringPort(data []byte) (net.Port, net.Port, error) {
	fmt.Println("in infra-conf-common.go func  parseJSONStringPort")
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return net.Port(0), net.Port(0), err
	}
	return parseStringPort(s)
}

type PortRange struct {
	From uint32
	To   uint32
}

func (v *PortRange) Build() *net.PortRange {
	fmt.Println("in infra-conf-common.go func (v *PortRange) Build()")
	return &net.PortRange{
		From: v.From,
		To:   v.To,
	}
}

// UnmarshalJSON implements encoding/json.Unmarshaler.UnmarshalJSON
func (v *PortRange) UnmarshalJSON(data []byte) error {
	fmt.Println("in infra-conf-common.go func (v *PortRange) UnmarshalJSON")
	port, err := parseIntPort(data)
	if err == nil {
		v.From = uint32(port)
		v.To = uint32(port)
		return nil
	}

	from, to, err := parseJSONStringPort(data)
	if err == nil {
		v.From = uint32(from)
		v.To = uint32(to)
		if v.From > v.To {
			return errors.New("invalid port range")
		}
		return nil
	}

	return errors.New("invalid port range")
}

type PortList struct {
	Range []PortRange
}

func (list *PortList) Build() *net.PortList {
	fmt.Println("in infra-conf-common.go func (list *PortList) Build()")
	portList := new(net.PortList)
	for _, r := range list.Range {
		portList.Range = append(portList.Range, r.Build())
	}
	return portList
}

// UnmarshalJSON implements encoding/json.Unmarshaler.UnmarshalJSON
func (list *PortList) UnmarshalJSON(data []byte) error {
	fmt.Println("in infra-conf-common.go func (list *PortList) UnmarshalJSON")
	var listStr string
	var number uint32
	if err := json.Unmarshal(data, &listStr); err != nil {
		if err2 := json.Unmarshal(data, &number); err2 != nil {
			return errors.New("invalid port")
		}
	}
	rangelist := strings.Split(listStr, ",")
	for _, rangeStr := range rangelist {
		trimmed := strings.TrimSpace(rangeStr)
		if len(trimmed) > 0 {
			if strings.Contains(trimmed, "-") {
				from, to, err := parseStringPort(trimmed)
				if err != nil {
					return errors.New("invalid port range")
				}
				list.Range = append(list.Range, PortRange{From: uint32(from), To: uint32(to)})
			} else {
				port, err := parseIntPort([]byte(trimmed))
				if err != nil {
					return errors.New("invalid port")
				}
				list.Range = append(list.Range, PortRange{From: uint32(port), To: uint32(port)})
			}
		}
	}
	if number != 0 {
		list.Range = append(list.Range, PortRange{From: number, To: number})
	}
	return nil
}

type User struct {
	EmailString string `json:"email"`
	LevelByte   byte   `json:"level"`
}

func (v *User) Build() *protocol.User {
	fmt.Println("in infra-conf-common.go func (v *User) Build()")
	return &protocol.User{
		Email: v.EmailString,
		Level: uint32(v.LevelByte),
	}
}
