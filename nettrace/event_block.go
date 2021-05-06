package nettrace

import (
	"bytes"
	"errors"
	"io"
	"unsafe"
)

// EventBlock contains a set of events.
type EventBlock struct {
	Header EventBlockHeader
	// Payload contains EventBlobs - serialized fragments which represent actual
	// events and metadata records.
	Payload *bytes.Buffer
}

func (b EventBlock) IsCompressed() bool {
	return b.Header.Flags&0x0001 != 0
}

type EventBlockHeader struct {
	// Size of the header including this field.
	Size  short
	Flags short
	// MinTimestamp specifies the minimum timestamp of any event in this block.
	MinTimestamp long
	// MinTimestamp specifies the maximum timestamp of any event in this block.
	MaxTimestamp long
	// Padding: optional reserved space to reach Size bytes.
}

const eventBlockHeaderLength = int32(unsafe.Sizeof(EventBlockHeader{}))

type EventBlob struct {
	Header EventBlobHeader
	sorted bool
	// TODO: keep Event and Metadata structures here or just Payload?
}

func (b EventBlob) IsSorted() bool { return b.sorted }

// EventBlobHeader used for both compressed and uncompressed formats.
type EventBlobHeader struct {
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

// EventBlockFromObject ...
// TODO: refactor API.
func EventBlockFromObject(o Object) (EventBlock, error) {
	p := parser{Reader: o.Payload}
	var b EventBlock
	p.read(&b.Header)
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

// Events ...
// TODO: refactor API:
//   func (b BlobBlock) Next(*Blob) error
func (b *EventBlock) Events() ([]EventBlob, error) {
	var fill func(*EventBlob) error
	if b.IsCompressed() {
		fill = b.readRecordHeaderCompressed
	} else {
		fill = b.readRecordHeader
	}

	events := make([]EventBlob, 0)
	var previousHeader EventBlobHeader

loop:
	for {
		e := EventBlob{Header: previousHeader}
		err := fill(&e)
		switch {
		case err == nil:
		case errors.Is(err, io.EOF):
			break loop
		default:
			return nil, err
		}

		// TODO: metadata processing.
		record := bytes.NewBuffer(b.Payload.Next(int(e.Header.PayloadSize)))
		var h MetadataHeader
		err = readMetadataHeader(record, &h)

		events = append(events, e)
		previousHeader = e.Header
	}

	return events, nil
}

func (b EventBlock) readRecordHeader(e *EventBlob) error {
	p := parser{Reader: b.Payload}
	p.read(e)
	// In the context of an EventBlock the low 31 bits are a foreign key to the
	// event's metadata. In the context of a metadata block the low 31 bits are
	// always zeroed. The high bit is the IsSorted flag.
	e.Header.MetadataID &= 0x7FFF
	e.sorted = uint32(e.Header.MetadataID)&0x8000 == 0
	return p.error()
}

func (b EventBlock) readRecordHeaderCompressed(e *EventBlob) error {
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
	return p.error()
}
