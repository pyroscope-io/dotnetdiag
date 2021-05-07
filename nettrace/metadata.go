package nettrace

import (
	"errors"
	"fmt"
	"io"

	"github.com/pyroscope-io/dotnetdiag/nettrace/typecode"
)

var ErrNotImplemented = errors.New("not implemented")

type Metadata struct {
	Header  MetadataHeader
	Payload *MetadataPayload
	p       *parser
}

type MetadataHeader struct {
	MetaDataID   int32
	ProviderName string
	EventID      int32
	EventName    string
	Keywords     long
	Version      MetadataVersion
	Level        int32
}

type MetadataVersion int32

const (
	_ MetadataVersion = iota

	MetadataLegacyV1 // Used by NetPerf version 1
	MetadataLegacyV2 // Used by NetPerf version 2
	MetadataNetTrace // Used by NetPerf (version 3) and NetTrace (version 4+)
)

type MetadataPayload struct {
	FieldCount int32
	Fields     []MetadataField
	Tags       []MetadataTag // V5 and later only.
}

type MetadataField struct {
	TypeCode typecode.TypeCode
	// ArrayTypeCode is an optional field only appears when TypeCode is Array.
	ArrayTypeCode typecode.TypeCode
	// For primitive types and strings Payload is not present, however if TypeCode is Object (1)
	// then Payload is another payload description (that is a field count, followed by a list of
	// field definitions). These can be nested to arbitrary depth.
	Payload *MetadataPayload
	Name    string
}

type MetadataTag struct {
	PayloadBytes int32
	Kind         MetadataTagKind
	OpCode       byte
	FieldCount   int32
}

type MetadataTagKind byte

const (
	_ MetadataTagKind = iota
	TagKindOpCode
	TagKindV2Params
)

func MetadataFromBlob(blob Blob) (*Metadata, error) {
	md := Metadata{
		Payload: new(MetadataPayload),
		p:       &parser{Buffer: blob.Payload},
	}
	md.p.read(&md.Header.MetaDataID)
	md.Header.ProviderName = md.p.utf16nts()
	md.p.read(&md.Header.EventID)
	md.Header.EventName = md.p.utf16nts()
	md.p.read(&md.Header.Keywords)
	md.p.read(&md.Header.Version)
	md.p.read(&md.Header.Level)

	if err := md.readMetadataPayload(md.Payload); err != nil {
		return nil, err
	}
	if _, err := blob.Payload.ReadByte(); err != io.EOF {
		return nil, fmt.Errorf("%w: V5 matadata payload", ErrNotImplemented)
	}
	if err := md.p.error(); err != nil {
		return nil, err
	}

	return &md, nil
}

func (md *Metadata) readMetadataPayload(mp *MetadataPayload) error {
	md.p.read(&mp.FieldCount)
	if mp.FieldCount == 0 {
		return md.p.error()
	}
	for i := int32(0); i < mp.FieldCount; i++ {
		var f MetadataField
		if err := md.readMetadataField(&f); err != nil {
			return err
		}
		mp.Fields = append(mp.Fields, f)
	}
	return md.p.error()
}

func (md *Metadata) readMetadataField(f *MetadataField) error {
	md.p.read(&f.TypeCode)
	switch f.TypeCode {
	default:
		// Built-in types do not have payload.
	case typecode.Array:
		return fmt.Errorf("%w: V5", ErrNotImplemented)
	case typecode.Object:
		if err := md.readMetadataPayload(f.Payload); err != nil {
			return err
		}
	}
	f.Name = md.p.utf16nts()
	return md.p.error()
}
