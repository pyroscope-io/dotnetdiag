package nettrace_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
	"github.com/pyroscope-io/dotnetdiag/nettrace/sampler"
)

func TestStream(t *testing.T) {
	sample, err := os.Open("testdata/golden.nettrace")
	requireNoError(t, err)
	expected, err := os.ReadFile("testdata/expected.nettrace")
	requireNoError(t, err)

	stream := nettrace.NewStream(sample)
	trace, err := stream.Open()
	requireNoError(t, err)

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
			r := newRenderer()
			s.Walk(r.visitor)
			var b bytes.Buffer
			r.dumpFlat(&b)
			if b.String() != string(expected) {
				t.Fatalf("Unexpected output")
			}
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

type renderer struct {
	out   map[string]time.Duration
	names []string
	val   time.Duration
	prev  int
}

func newRenderer() *renderer {
	return &renderer{out: make(map[string]time.Duration)}
}

func (r *renderer) visitor(frame sampler.FrameInfo) {
	if frame.Level > r.prev || (frame.Level == 0 && r.prev == 0) {
		r.names = append(r.names, frame.Name)
	} else {
		r.complete()
		if frame.Level == 0 {
			r.names = []string{frame.Name}
		} else {
			r.names = append(r.names[:frame.Level], frame.Name)
		}
	}
	r.val = frame.SampledTime
	r.prev = frame.Level
}

func (r *renderer) complete() {
	if len(r.names) > 0 {
		r.out[strings.Join(r.names, ";")] += r.val
	}
}

func (r *renderer) dumpFlat(w io.Writer) {
	r.complete()
	s := make([]string, 0, len(r.out))
	for k, v := range r.out {
		s = append(s, fmt.Sprint(k, " ", v.Nanoseconds()))
	}
	sort.Strings(s)
	for _, x := range s {
		_, _ = fmt.Fprintln(w, x)
	}
}

func (r *renderer) dumpTree(w io.Writer) func(sampler.FrameInfo) {
	return func(frame sampler.FrameInfo) {
		_, _ = fmt.Fprintf(w, "%s(%v) %s\n", padding(frame.Level), frame.SampledTime, frame.Name)
	}
}

func padding(x int) string {
	var s strings.Builder
	for i := 0; i < x; i++ {
		s.WriteString("\t")
	}
	return s.String()
}
