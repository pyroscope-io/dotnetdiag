package nettrace

import (
	"io"

	"github.com/pyroscope-io/dotnetdiag/nettrace/typecode"
)

type Metadata struct {
	Header MetadataHeader
}

type MetadataHeader struct {
	MetaDataID   int32  // The Meta-Data ID that is being defined.
	ProviderName string // The 2 byte Unicode, null terminated string representing the Name of the Provider (e.g. EventSource)
	EventID      int32  // A small number that uniquely represents this Event within this provider.
	EventName    string // The 2 byte Unicode, null terminated string representing the Name of the Event
	Keywords     long   // 64 bit set of groups (keywords) that this event belongs to.
	Version      int32  // The version number for this event.
	Level        int32  // The verbosity (5 is verbose, 1 is only critical) for the event.
}

type MetadataPayload struct {
	FieldCount int32
	Fields     []MetadataField
	Tags       []MetadataTag // V5 and later only.
}

type MetadataField struct {
	Type typecode.Typecode
	// ArrayTypeCode is an optional field only appears when Type is EventPipeTypeCodeArray.
	ArrayTypeCode typecode.Typecode
	// For primitive types and strings Payload is not present, however if Type is Object (1)
	// then Payload is another payload description (that is a field count, followed by a list of
	// field definitions). These can be nested to arbitrary depth.
	Payload *MetadataPayload
	Name    string
}

type MetadataTag struct {
	PayloadBytes int32
	Kind         MetadataTagKind
	OpCode       byte
	V2FieldCount int32
}

type MetadataTagKind byte

const (
	_ MetadataTagKind = iota
	TagKindOpCode
	TagKindV2Params
)

func readMetadataHeader(r io.Reader, h *MetadataHeader) error {
	p := parser{Reader: r}
	p.read(&h.MetaDataID)
	h.ProviderName = p.utf16nts()
	p.read(&h.EventID)
	h.EventName = p.utf16nts()
	p.read(&h.Keywords)
	p.read(&h.Version)
	p.read(&h.Level)
	return p.error()
}
