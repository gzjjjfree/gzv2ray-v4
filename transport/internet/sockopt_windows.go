package internet

import (
	"fmt"
	//"os/exec"
	"syscall"
)

const (
	TCP_FASTOPEN = 15 // nolint: golint,stylecheck
)

func setTFO(fd syscall.Handle, settings SocketConfig_TCPFastOpenState) error {
	switch settings {
	case SocketConfig_Enable:
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_FASTOPEN, 1); err != nil {
			return err
		}
	case SocketConfig_Disable:
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_FASTOPEN, 0); err != nil {
			return err
		}
	}
	return nil
}

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		if err := setTFO(syscall.Handle(fd), config.Tfo); err != nil {
			fmt.Println("errors in address: ", address)
			return err
		}

	}

	return nil
}

func applyInboundSocketOptions(network string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		if err := setTFO(syscall.Handle(fd), config.Tfo); err != nil {
			return err
		}
	}

	return nil
}

func bindAddr(fd uintptr, ip []byte, port uint32) error {
	fmt.Println("errors in : ", fd, ip, port)
	return nil
}

func setReuseAddr(fd uintptr) error {
	fmt.Println("in transport-interrnet-sockopt_windows.go func setReuseAddr")
	//fmt.Println("errors in address: ", fd)
	return nil
}

func setReusePort(fd uintptr) error {
	fmt.Println("in transport-interrnet-sockopt_windows.go func setReusePort")
	//setReuseAddr(fd)
	//fmt.Println("errors in address: ", fd)
	return nil
}
