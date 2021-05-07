package nettrace_test

import (
	"errors"
	"io"
	"os"
	"runtime/debug"
	"testing"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
)

type sample struct {
	*os.File
	*nettrace.Decoder
}

func TestSample(t *testing.T) {
	requireSample(t)
}

func TestDecode(t *testing.T) {
	s := requireSample(t)
	var o nettrace.Object
	for {
		err := s.Decode(&o)
		switch {
		case err == nil:
		case errors.Is(err, io.EOF):
			return
		default:
			requireNoError(t, err)
		}

		switch o.Type {
		case nettrace.ObjectTypeSPBlock:
			block, err := nettrace.SequencePointBlockFromObject(o)
			requireNoError(t, err)
			t.Logf("SP block: %d; %+v", block.TimeStamp, block.Threads)

		case nettrace.ObjectTypeStackBlock:
			block, err := nettrace.StackBlockFromObject(o)
			requireNoError(t, err)
			t.Logf("Stack block: %d, %d", block.FirstID, len(block.Stacks))

		case nettrace.ObjectTypeEventBlock:
			block, err := nettrace.BlobBlockFromObject(o)
			requireNoError(t, err)
			readEventBlock(t, block)

		case nettrace.ObjectTypeMetadataBlock:
			block, err := nettrace.BlobBlockFromObject(o)
			requireNoError(t, err)
			readMetadataBlock(t, block)

		default:
			t.Fatalf("Unexpected object type %q", o.Type)
		}
	}
}

func readEventBlock(t *testing.T, block *nettrace.BlobBlock) {
	t.Helper()
	for {
		var blob nettrace.Blob
		err := block.Next(&blob)
		switch {
		case errors.Is(err, io.EOF):
			return
		case err == nil:
			// t.Logf("Event: (%d) %#v", blob.Payload.Len(), blob.Header.MetadataID)
		default:
			requireNoError(t, err)
		}
	}
}

func readMetadataBlock(t *testing.T, block *nettrace.BlobBlock) {
	t.Helper()
	var blob nettrace.Blob
	for {
		err := block.Next(&blob)
		switch {
		case errors.Is(err, io.EOF):
			return
		case err == nil:
			md, err := nettrace.MetadataFromBlob(blob)
			requireNoError(t, err)
			t.Logf("Metadata block (%d): %+v; %+v", blob.Header.PayloadSize, md.Header, md.Payload)
		default:
			requireNoError(t, err)
		}
	}
}

func requireSample(t *testing.T) *sample {
	t.Helper()
	file, err := os.Open("testdata/golden.nettrace")
	requireNoError(t, err)
	dec := nettrace.NewDecoder(file)
	trace, err := dec.OpenTrace()
	requireNoError(t, err)
	t.Logf("Trace object: %#v", trace)
	return &sample{
		File:    file,
		Decoder: dec,
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s\n", err, string(debug.Stack()))
	}
}
