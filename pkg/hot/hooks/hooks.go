package hooks

import (
	"encoding/binary"
	"errors"
	"gateway/pkg/interfaces"
)

var (
	Version    = "0.0.1"
	HookHeader = interfaces.HookHeader(hookHeader)
	HookBody   = interfaces.HookBody(hookBody)

	minID   uint16 = 1000
	maxID   uint16 = 60000
	minSize uint32 = 10              // 10 字节
	maxSize uint32 = 1024 * 1024 * 1 // 1 兆
)

var (
	ErrBadHeaderSize = errors.New("bad header size")
	ErrBadHeaderID   = errors.New("bad header id")
)

func hookHeader(agent interfaces.Agent, header []byte) error {
	if header != nil {
		id := binary.LittleEndian.Uint16(header[:2])
		size := binary.LittleEndian.Uint32(header[2:6])
		crc := binary.LittleEndian.Uint32(header[6:10])
		_ = crc

		if id < minID || id > maxID {
			return ErrBadHeaderID
		}

		if size < minSize || size > maxSize {
			return ErrBadHeaderSize
		}
	}

	return nil
}

func hookBody(agent interfaces.Agent, header []byte, body []byte) error {
	if body != nil {
		// TODO
	}

	return nil
}
