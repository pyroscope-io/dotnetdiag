package nettrace

import (
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"
)

// TODO: refactor
//  use *byte.Buffer instead of io.Reader
//  improve utf16 NTS
//  reflect for struct?
type parser struct {
	io.Reader
	errs []error
}

func (p *parser) error() error {
	if len(p.errs) != 0 {
		return fmt.Errorf("parser: %w", p.errs[0])
	}
	return nil
}

func (p *parser) read(v interface{}) {
	if err := binary.Read(p, binary.LittleEndian, v); err != nil {
		p.errs = append(p.errs, err)
	}
}

func (p *parser) uvarint() uint64 {
	n, err := binary.ReadUvarint(p.Reader.(io.ByteReader))
	if err != nil {
		p.errs = append(p.errs, err)
	}
	return n
}

func (p *parser) utf16nts() string {
	s := make([]uint16, 0, 64)
	var c uint16
	for {
		if p.errs != nil {
			return ""
		}
		p.read(&c)
		if c == 0x0 {
			break
		}
		s = append(s, c)
	}
	return string(utf16.Decode(s))
}
