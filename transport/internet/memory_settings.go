package internet

import "fmt"

// MemoryStreamConfig is a parsed form of StreamConfig. This is used to reduce number of Protobuf parsing.
type MemoryStreamConfig struct {
	ProtocolName     string
	ProtocolSettings interface{}
	SecurityType     string
	SecuritySettings interface{}
	SocketSettings   *SocketConfig
}

// ToMemoryStreamConfig converts a StreamConfig to MemoryStreamConfig. It returns a default non-nil MemoryStreamConfig for nil input.
// ToMemoryStreamConfig 将 StreamConfig 转换为 MemoryStreamConfig。它为 nil 输入返回默认的非 nil MemoryStreamConfig。
func ToMemoryStreamConfig(s *StreamConfig) (*MemoryStreamConfig, error) {
	fmt.Println("in transport-internet-memory_settings.go func ToMemoryStreamConfig(s *StreamConfig)")
	ets, err := s.GetEffectiveTransportSettings()
	if err != nil {
		return nil, err
	}

	mss := &MemoryStreamConfig{
		ProtocolName:     s.GetEffectiveProtocol(),
		ProtocolSettings: ets,
	}

	if s != nil {
		mss.SocketSettings = s.SocketSettings
	}

	if s != nil && s.HasSecuritySettings() {
		ess, err := s.GetEffectiveSecuritySettings()
		if err != nil {
			return nil, err
		}
		mss.SecurityType = s.SecurityType
		mss.SecuritySettings = ess
	}

	return mss, nil
}
