package nettrace_test

import (
	"io"
	"os"
	"runtime/debug"
	"testing"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
)

func TestStream(t *testing.T) {
	file, err := os.Open("testdata/golden.nettrace")
	requireNoError(t, err)

	stream := nettrace.NewStream(file)
	trace, err := stream.Open()
	requireNoError(t, err)
	t.Logf("Trace: %+v", trace)

	stream.EventHandler = func(e *nettrace.Blob) error {
		// t.Logf("Event %+v; payload size %d", e.Header, e.Payload.Len())
		return nil
	}

	stream.MetadataHandler = func(md *nettrace.Metadata) error {
		t.Logf("Metadata: %+v; %+v", md.Header, md.Payload)
		return nil
	}

	stream.StackBlockHandler = func(sb *nettrace.StackBlock) error {
		t.Logf("StackBlock %d; stacks: %d", sb.FirstID, len(sb.Stacks))
		return nil
	}

	stream.SequencePointBlockHandler = func(spb *nettrace.SequencePointBlock) error {
		t.Logf("SequencePointBlock %d; %+v", spb.TimeStamp, spb.Threads)
		return nil
	}

	for {
		err = stream.Next()
		switch err {
		default:
			requireNoError(t, err)
		case io.EOF:
			return
		case nil:
			continue
		}
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s\n", err, string(debug.Stack()))
	}
}
