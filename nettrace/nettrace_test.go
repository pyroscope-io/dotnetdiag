package nettrace_test

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
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

	s := sampler.NewCPUTimeSampler(trace)
	stream.EventHandler = s.EventHandler
	stream.MetadataHandler = s.MetadataHandler
	stream.StackBlockHandler = s.StackBlockHandler
	stream.SequencePointBlockHandler = s.SequencePointBlockHandler

	for {
		err = stream.Next()
		switch err {
		default:
			requireNoError(t, err)
		case nil:
			continue
		case io.EOF:
			s.WalkThread(4053210, func(frame sampler.FrameInfo) {
				if frame.SampledTime > 0 {
					fmt.Printf("%s(%s) %s\n", padding(frame.Level), frame.SampledTime, frame.Name)
				}
			})
			return
		}
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s\n", err, string(debug.Stack()))
	}
}

func padding(x int) string {
	var s strings.Builder
	for i := 0; i < x; i++ {
		s.WriteString("\t")
	}
	return s.String()
}
