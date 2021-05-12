package nettrace_test

import (
	"io"
	"os"
	"runtime/debug"
	"testing"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
	"github.com/pyroscope-io/dotnetdiag/nettrace/sampler"
)

func TestStream(t *testing.T) {
	file, err := os.Open("testdata/golden.nettrace")
	requireNoError(t, err)

	stream := nettrace.NewStream(file)
	trace, err := stream.Open()
	requireNoError(t, err)
	t.Logf("Trace: %+v", trace)

	s := sampler.NewSampler(trace)
	stream.EventHandler = s.EventHandler
	stream.MetadataHandler = s.MetadataHandler
	stream.StackBlockHandler = s.StackBlockHandler
	stream.SequencePointBlockHandler = s.SequencePointBlockHandler

	for {
		err = stream.Next()
		switch err {
		default:
			requireNoError(t, err)
		case io.EOF:
			sampler.Print(os.Stdout, s)
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
