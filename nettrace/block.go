package nettrace

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf16"
	"unsafe"
)

// BlobBlock contains a set of Blobs.
type BlobBlock struct {
	Header BlobBlockHeader
	// Payload contains Blobs - serialized fragments which represent
	// actual events and metadata records.
	Payload *bytes.Buffer

	compressed bool
	// lastHeader holds the most recent Blob header: this is used if the
	// block uses compressed headers format.
	lastHeader    BlobHeader
	extractHeader func(*Blob) error
	*parser
}

type BlobBlockHeader struct {
	// Size of the header including this field.
	Size  short
	Flags short
	// MinTimestamp specifies the minimum timestamp of any event in this block.
	MinTimestamp long
	// MinTimestamp specifies the maximum timestamp of any event in this block.
	MaxTimestamp long
	// Padding: optional reserved space to reach Size bytes.
}

const eventBlockHeaderLength = int32(unsafe.Sizeof(BlobBlockHeader{}))

type Blob struct {
	Header BlobHeader
	// Payload contains Event and Metadata records.
	Payload *bytes.Buffer
	sorted  bool
}

// BlobHeader used for both compressed and uncompressed blobs.
type BlobHeader struct {
	// EventSize specifies record size not counting this field.
	EventSize         int32
	MetadataID        int32
	SequenceNumber    int32
	ThreadID          long
	CaptureThreadID   long
	CaptureProcNumber int32
	StackID           int32
	TimeStamp         long
	ActivityID        guid
	RelatedActivityID guid
	PayloadSize       int32
}

type StackBlock struct {
	FirstID int32
	Stacks  []Stack
}

type Stack []byte

type SequencePointBlock struct {
	TimeStamp long
	Threads   []Thread
}

type Thread struct {
	ThreadID       long
	SequenceNumber int32
}

type compressedHeaderFlag byte

// The section lists compressed header flags: if otherwise specified, every
// flag is set when an appropriate header member should be read from stream.
const (
	flagMetadataID compressedHeaderFlag = 1 << iota
	// Specifies that CaptureThreadID, CaptureProcNumber and SequenceNumber
	// values are to be read from the stream: Incremented SequenceNumber should
	// be added to the previous value. If the flag is not set, and MetadataID
	// field is zero, incremented previous SequenceNumber should be used.
	flagCaptureThreadAndSequence
	flagThreadID
	flagStackID
	flagActivityID
	flagRelatedActivityID
	// Specifies that the events are sorted.
	flagIsSorted
	flagPayloadSize
)

func (b *BlobBlock) IsCompressed() bool { return b.compressed }

func (b *Blob) IsSorted() bool { return b.sorted }

func BlobBlockFromObject(o Object) (*BlobBlock, error) {
	p := parser{Buffer: o.Payload}
	var b BlobBlock
	p.read(&b.Header)
	b.compressed = b.Header.Flags&0x0001 != 0
	if b.compressed {
		b.extractHeader = b.readBlobHeaderCompressed
	} else {
		b.extractHeader = b.readBlobHeader
	}
	// Skip header padding.
	padLen := int32(b.Header.Size) - eventBlockHeaderLength
	if padLen > 0 {
		o.Payload.Next(int(padLen))
	}
	// Given that blocks are to be processed sequentially,
	// there is no need for a copy.
	b.Payload = o.Payload
	b.parser = &p
	return &b, p.error()
}

func (b *BlobBlock) Next(blob *Blob) error {
	err := b.extractHeader(blob)
	switch {
	default:
	case errors.Is(err, io.EOF):
		return io.EOF
	case err != nil:
		return err
	}
	pb := b.Payload.Next(int(blob.Header.PayloadSize))
	blob.Payload = bytes.NewBuffer(pb)
	return nil
}

func (b *BlobBlock) readBlobHeader(blob *Blob) error {
	b.read(blob)
	// In the context of an EventBlock the low 31 bits are a foreign key to the
	// event's metadata. In the context of a metadata block the low 31 bits are
	// always zeroed. The high bit is the IsSorted flag.
	blob.Header.MetadataID &= 0x7FFF
	blob.sorted = uint32(blob.Header.MetadataID)&0x8000 == 0
	return b.error()
}

func (b *BlobBlock) readBlobHeaderCompressed(blob *Blob) error {
	blob.Header = b.lastHeader
	var flags compressedHeaderFlag
	b.read(&flags)
	blob.sorted = flags&flagIsSorted != 0
	if flags&flagMetadataID != 0 {
		blob.Header.MetadataID = int32(b.uvarint())
	}
	if flags&flagCaptureThreadAndSequence != 0 {
		blob.Header.SequenceNumber = int32(b.uvarint()) + 1
		blob.Header.CaptureThreadID = long(b.uvarint())
		blob.Header.CaptureProcNumber = int32(b.uvarint())
	} else if blob.Header.MetadataID != 0 {
		blob.Header.SequenceNumber++
	}
	if flags&flagThreadID != 0 {
		blob.Header.ThreadID = long(b.uvarint())
	}
	if flags&flagStackID != 0 {
		blob.Header.StackID = int32(b.uvarint())
	}
	blob.Header.TimeStamp += long(b.uvarint())
	if flags&flagActivityID != 0 {
		b.read(&blob.Header.ActivityID)
	}
	if flags&flagRelatedActivityID != 0 {
		b.read(&blob.Header.RelatedActivityID)
	}
	if flags&flagPayloadSize != 0 {
		blob.Header.PayloadSize = int32(b.uvarint())
	}
	b.lastHeader = blob.Header
	return b.error()
}

func StackBlockFromObject(o Object) (*StackBlock, error) {
	var b StackBlock
	var count, size int32
	p := parser{Buffer: o.Payload}
	p.read(&b.FirstID)
	p.read(&count)
	b.Stacks = make([]Stack, count)
	for i := int32(0); i < count; i++ {
		p.read(&size)
		b.Stacks[i] = o.Payload.Next(int(size))
	}
	return &b, p.error()
}

func SequencePointBlockFromObject(o Object) (*SequencePointBlock, error) {
	var b SequencePointBlock
	var count int32
	p := parser{Buffer: o.Payload}
	p.read(&b.TimeStamp)
	p.read(&count)
	b.Threads = make([]Thread, count)
	for i := int32(0); i < count; i++ {
		var t Thread
		p.read(&t)
		b.Threads[i] = t
	}
	return &b, p.error()
}

// TODO: refactor
type parser struct {
	*bytes.Buffer
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
	n, err := binary.ReadUvarint(p)
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
