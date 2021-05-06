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
	// Payload contains Blobs - serialized fragments which represent
	// actual events and metadata records.
	Payload *bytes.Buffer

	compressed bool
	// lastHeader holds the most recent Blob header: this is used if the
	// block uses compressed headers format.
	lastHeader    BlobHeader
	extractHeader func(*Blob) error
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
	p := parser{Reader: o.Payload}
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
	p := parser{Reader: b.Payload}
	p.read(blob)
	// In the context of an EventBlock the low 31 bits are a foreign key to the
	// event's metadata. In the context of a metadata block the low 31 bits are
	// always zeroed. The high bit is the IsSorted flag.
	blob.Header.MetadataID &= 0x7FFF
	blob.sorted = uint32(blob.Header.MetadataID)&0x8000 == 0
	return p.error()
}

func (b *BlobBlock) readBlobHeaderCompressed(blob *Blob) error {
	blob.Header = b.lastHeader
	p := parser{Reader: b.Payload}
	var flags compressedHeaderFlag
	p.read(&flags)
	blob.sorted = flags&flagIsSorted != 0
	if flags&flagMetadataID != 0 {
		blob.Header.MetadataID = int32(p.uvarint())
	}
	if flags&flagCaptureThreadAndSequence != 0 {
		blob.Header.SequenceNumber = int32(p.uvarint()) + 1
		blob.Header.CaptureThreadID = long(p.uvarint())
		blob.Header.CaptureProcNumber = int32(p.uvarint())
	} else if blob.Header.MetadataID != 0 {
		blob.Header.SequenceNumber++
	}
	if flags&flagThreadID != 0 {
		blob.Header.ThreadID = long(p.uvarint())
	}
	if flags&flagStackID != 0 {
		blob.Header.StackID = int32(p.uvarint())
	}
	blob.Header.TimeStamp += long(p.uvarint())
	if flags&flagActivityID != 0 {
		p.read(&blob.Header.ActivityID)
	}
	if flags&flagRelatedActivityID != 0 {
		p.read(&blob.Header.RelatedActivityID)
	}
	if flags&flagPayloadSize != 0 {
		blob.Header.PayloadSize = int32(p.uvarint())
	}
	b.lastHeader = blob.Header
	return p.error()
}
