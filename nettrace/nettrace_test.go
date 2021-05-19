package nettrace_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"testing"
	"time"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
	"github.com/pyroscope-io/dotnetdiag/nettrace/profiler"
)

func TestNetTraceDecoding(t *testing.T) {
	t.Run(".Net 5.0 SampleProfiler", func(t *testing.T) {
		t.Run("Web app", func(t *testing.T) {
			t.Run("Managed code only", func(t *testing.T) {
				requireEqual(t,
					"testdata/dotnet-5.0-SampleProfiler-webapp.golden.nettrace",
					"testdata/dotnet-5.0-SampleProfiler-webapp-managed-only.txt",
					profiler.WithManagedCodeOnly())
			})
			t.Run("Managed and native code", func(t *testing.T) {
				requireEqual(t,
					"testdata/dotnet-5.0-SampleProfiler-webapp.golden.nettrace",
					"testdata/dotnet-5.0-SampleProfiler-webapp.txt")
			})
		})

		t.Run("Simple single thread app", func(t *testing.T) {
			t.Run("Managed code only", func(t *testing.T) {
				requireEqual(t,
					"testdata/dotnet-5.0-SampleProfiler-single-thread.golden.nettrace",
					"testdata/dotnet-5.0-SampleProfiler-single-thread-managed-only.txt",
					profiler.WithManagedCodeOnly())
			})
			t.Run("Managed and native code", func(t *testing.T) {
				requireEqual(t,
					"testdata/dotnet-5.0-SampleProfiler-single-thread.golden.nettrace",
					"testdata/dotnet-5.0-SampleProfiler-single-thread.txt")
			})
		})
	})
}

func requireEqual(t *testing.T, sample, expected string, options ...profiler.Option) {
	t.Helper()

	s, err := os.Open(sample)
	requireNoError(t, err)
	e, err := os.ReadFile(expected)
	requireNoError(t, err)

	stream := nettrace.NewStream(s)
	trace, err := stream.Open()
	requireNoError(t, err)

	p := profiler.NewSampleProfiler(trace, options...)
	stream.EventHandler = p.EventHandler
	stream.MetadataHandler = p.MetadataHandler
	stream.StackBlockHandler = p.StackBlockHandler
	stream.SequencePointBlockHandler = p.SequencePointBlockHandler

	for {
		err = stream.Next()
		switch err {
		default:
			requireNoError(t, err)
		case nil:
			continue
		case io.EOF:
			var b bytes.Buffer
			dump(&b, p.Samples())
			if b.String() != string(e) {
				t.Fatal("output mismatch")
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

func dump(w io.Writer, samples map[string]time.Duration) {
	names := make([]string, 0, len(samples))
	for k := range samples {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		_, _ = fmt.Fprintln(w, n, samples[n].Nanoseconds())
	}
}
