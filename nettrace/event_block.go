package nettrace

import (
	"bytes"
	"errors"
	"io"
	"unsafe"
)

// BlobBlock contains a set of Blobs.
type BlobBlock struct {
	Header BlobBlockHeader
	// Payload contains EventBlobs - serialized fragments which represent
	// actual events and metadata records.
	Payload *bytes.Buffer

	compressed bool
	// lastHeader holds the most recent Blob header: this is used if the
	// block uses compressed headers format.
	lastHeader BlobHeader
	extract    func(*Blob) error
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
	// Payload contains Event and Metadata record.
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

func (b BlobBlock) IsCompressed() bool { return b.compressed }

func (b Blob) IsSorted() bool { return b.sorted }

func BlobBlockFromObject(o Object) (BlobBlock, error) {
	p := parser{Reader: o.Payload}
	var b BlobBlock
	p.read(&b.Header)
	b.compressed = b.Header.Flags&0x0001 != 0
	if b.compressed {
		b.extract = b.readRecordHeaderCompressed
	} else {
		b.extract = b.readRecordHeader
	}
	// Skip header padding.
	padLen := int32(b.Header.Size) - eventBlockHeaderLength
	if padLen > 0 {
		o.Payload.Next(int(padLen))
	}
	// Given that blocks are to be processed sequentially, there is no
	// need for a re-slicing the underlying buffer.
	b.Payload = o.Payload
	return b, p.error()
}

func (b *BlobBlock) Next(e *Blob) error {
	err := b.extract(e)
	switch {
	default:
	case errors.Is(err, io.EOF):
		return io.EOF
	case err != nil:
		return err
	}
	pb := b.Payload.Next(int(e.Header.PayloadSize))
	e.Payload = bytes.NewBuffer(pb)
	return nil
}

func (b *BlobBlock) readRecordHeader(e *Blob) error {
	p := parser{Reader: b.Payload}
	p.read(e)
	// In the context of an EventBlock the low 31 bits are a foreign key to the
	// event's metadata. In the context of a metadata block the low 31 bits are
	// always zeroed. The high bit is the IsSorted flag.
	e.Header.MetadataID &= 0x7FFF
	e.sorted = uint32(e.Header.MetadataID)&0x8000 == 0
	return p.error()
}

func (b *BlobBlock) readRecordHeaderCompressed(e *Blob) error {
	e.Header = b.lastHeader
	p := parser{Reader: b.Payload}
	var flags compressedHeaderFlag
	p.read(&flags)
	e.sorted = flags&flagIsSorted != 0
	if flags&flagMetadataID != 0 {
		e.Header.MetadataID = int32(p.uvarint())
	}
	if flags&flagCaptureThreadAndSequence != 0 {
		e.Header.SequenceNumber = int32(p.uvarint()) + 1
		e.Header.CaptureThreadID = long(p.uvarint())
		e.Header.CaptureProcNumber = int32(p.uvarint())
	} else if e.Header.MetadataID != 0 {
		e.Header.SequenceNumber++
	}
	if flags&flagThreadID != 0 {
		e.Header.ThreadID = long(p.uvarint())
	}
	if flags&flagStackID != 0 {
		e.Header.StackID = int32(p.uvarint())
	}
	e.Header.TimeStamp += long(p.uvarint())
	if flags&flagActivityID != 0 {
		p.read(&e.Header.ActivityID)
	}
	if flags&flagRelatedActivityID != 0 {
		p.read(&e.Header.RelatedActivityID)
	}
	if flags&flagPayloadSize != 0 {
		e.Header.PayloadSize = int32(p.uvarint())
	}
	b.lastHeader = e.Header
	return p.error()
}
