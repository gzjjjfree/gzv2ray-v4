package buf

import (
	"io"
	"net"
	"os"
	"syscall"
	"time"
	"errors"
	"fmt"
)

// Reader extends io.Reader with MultiBuffer.
// Reader 使用 MultiBuffer 扩展了 io.Reader。
type Reader interface {
	// ReadMultiBuffer reads content from underlying reader, and put it into a MultiBuffer.
	// ReadMultiBuffer 从底层读取器读取内容，并将其放入 MultiBuffer 中。
	ReadMultiBuffer() (MultiBuffer, error)
}

// ErrReadTimeout is an error that happens with IO timeout.
// ErrReadTimeout 是因 IO 超时而发生的错误。
var ErrReadTimeout = errors.New("IO timeout")

// TimeoutReader is a reader that returns error if Read() operation takes longer than the given timeout.
// TimeoutReader 是一个读取器，如果 Read() 操作花费的时间超过给定的超时时间，则返回错误。
type TimeoutReader interface {
	ReadMultiBufferTimeout(time.Duration) (MultiBuffer, error)
}

// Writer extends io.Writer with MultiBuffer.
// Writer 使用 MultiBuffer 扩展了 io.Writer。
type Writer interface {
	// WriteMultiBuffer writes a MultiBuffer into underlying writer.
	// WriteMultiBuffer 将 MultiBuffer 写入底层写入器。
	WriteMultiBuffer(MultiBuffer) error
}

// WriteAllBytes ensures all bytes are written into the given writer.
// WriteAllBytes 确保所有字节都写入给定的写入器
func WriteAllBytes(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		n, err := writer.Write(payload)
		if err != nil {
			return err
		}
		payload = payload[n:]
	}
	return nil
}

func isPacketReader(reader io.Reader) bool {
	_, ok := reader.(net.PacketConn)
	return ok
}

// NewReader creates a new Reader.
// NewReader 创建一个新的阅读器。
// The Reader instance doesn't take the ownership of reader.
// Reader 实例不拥有 reader 的所有权。
func NewReader(reader io.Reader) Reader {
	fmt.Println("in common-buf-io.go func NewReader ")
	if mr, ok := reader.(Reader); ok {
		return mr
	}

	if isPacketReader(reader) {
		return &PacketReader{
			Reader: reader,
		}
	}

	_, isFile := reader.(*os.File)
	if !isFile && useReadv {
		if sc, ok := reader.(syscall.Conn); ok {
			rawConn, err := sc.SyscallConn()
			if err != nil {
				fmt.Println("failed to get sysconn")
			} else {
				return NewReadVReader(reader, rawConn)
			}
		}
	}

	return &SingleReader{
		Reader: reader,
	}
}

// NewPacketReader creates a new PacketReader based on the given reader.
func NewPacketReader(reader io.Reader) Reader {
	if mr, ok := reader.(Reader); ok {
		return mr
	}

	return &PacketReader{
		Reader: reader,
	}
}

func isPacketWriter(writer io.Writer) bool {
	if _, ok := writer.(net.PacketConn); ok {
		return true
	}

	// If the writer doesn't implement syscall.Conn, it is probably not a TCP connection.
	if _, ok := writer.(syscall.Conn); !ok {
		return true
	}
	return false
}

// NewWriter creates a new Writer.
func NewWriter(writer io.Writer) Writer {
	if mw, ok := writer.(Writer); ok {
		return mw
	}

	if isPacketWriter(writer) {
		return &SequentialWriter{
			Writer: writer,
		}
	}

	return &BufferToBytesWriter{
		Writer: writer,
	}
}
