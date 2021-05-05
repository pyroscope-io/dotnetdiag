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
		case errors.Is(err, io.EOF):
			return
		case err == nil:
			t.Logf("%s: %#v", o.Type, o)
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
