package nettrace

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

type Parser struct {
	*bytes.Buffer
	errs []error
}

func (p *Parser) Err() error {
	if len(p.errs) != 0 {
		return fmt.Errorf("parser: %w", p.errs[0])
	}
	return nil
}

func (p *Parser) Read(v interface{}) {
	if err := binary.Read(p.Buffer, binary.LittleEndian, v); err != nil {
		p.errs = append(p.errs, err)
	}
}

func (p *Parser) Uvarint() uint64 {
	n, err := binary.ReadUvarint(p)
	if err != nil {
		p.errs = append(p.errs, err)
	}
	return n
}

func (p *Parser) UTF16NTS() string {
	s := make([]uint16, 0, 64)
	var c uint16
	for {
		if p.errs != nil {
			return ""
		}
		p.Read(&c)
		if c == 0x0 {
			break
		}
		s = append(s, c)
	}
	return string(utf16.Decode(s))
}
