package writer

import (
	"encoding/binary"
	"io"
	"sync"
)

type Writer struct {
	w   io.Writer
	b   []byte
	err error
	sync.Mutex
}

func New(wr io.Writer) *Writer {
	w := &Writer{
		w: wr,
	}

	return w
}

func (w *Writer) Write(msgID uint16, msg []byte, seqID uint32) error {
	if w.err != nil {
		return w.err
	}

	length := len(msg)
	buf := make([]byte, length+10)
	copy(buf[10:], msg)

	binary.LittleEndian.PutUint16(buf[:2], msgID)
	binary.LittleEndian.PutUint32(buf[2:6], uint32(length+10))
	binary.LittleEndian.PutUint32(buf[6:10], seqID)

	w.Lock()
	defer w.Unlock()
	w.b = append(w.b, buf...)

	return nil
}

func (w *Writer) Pop() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}

	// TODO 加pool 优化
	w.Lock()
	defer w.Unlock()
	tmp := make([]byte, len(w.b))
	copy(tmp, w.b)

	if cap(w.b) > 262144 {
		w.b = nil
	} else {
		w.b = w.b[:0]
	}

	return tmp, nil
}

func (w *Writer) Flush(b []byte) error {
	if w.err != nil {
		return w.err
	}

	if _, err := w.w.Write(b); err != nil {
		w.err = err
	}

	return w.err
}
