package pipe

import (
	"fmt"
	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
)

// Writer is a buf.Writer that writes data into a pipe.
type Writer struct {
	pipe *pipe
}

// WriteMultiBuffer implements buf.Writer.
func (w *Writer) WriteMultiBuffer(mb buf.MultiBuffer) error {
	return w.pipe.WriteMultiBuffer(mb)
}

// Close implements io.Closer. After the pipe is closed, writing to the pipe will return io.ErrClosedPipe, while reading will return io.EOF.
func (w *Writer) Close() error {
	fmt.Println("in transport-pipe-writer.go func (w *Writer) Close()")
	return w.pipe.Close()
}

// Interrupt implements common.Interruptible.
func (w *Writer) Interrupt() {
	w.pipe.Interrupt()
}