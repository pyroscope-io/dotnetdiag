package nettrace_test

import (
	"encoding/json"
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

	var counters = &struct {
		Metas   map[int32]int
		Threads map[int64]int
		Stacks  map[int32]int
	}{
		Metas:   make(map[int32]int),
		Threads: make(map[int64]int),
		Stacks:  make(map[int32]int),
	}
	defer func() {
		b, _ := json.Marshal(counters)
		t.Logf(string(b))
	}()

	stacks := make(map[int32][]byte)
	metas := make(map[int32]*nettrace.Metadata)

	stream.EventHandler = func(e *nettrace.Blob) error {
		counters.Metas[e.Header.MetadataID]++
		counters.Threads[int64(e.Header.ThreadID)]++
		counters.Stacks[e.Header.StackID]++

		switch e.Header.MetadataID {
		case 6, 7:
			event, err := nettrace.ParseMethodLoadUnloadTraceData(e.Payload)
			requireNoError(t, err)
			// if strings.Contains(event.MethodNamespace, "Razor") { }
			stack := stacks[e.Header.StackID]
			_ = stack
			meta := metas[e.Header.MetadataID]
			_ = meta
			t.Log("Event:", event)
		}

		return nil
	}

	stream.MetadataHandler = func(md *nettrace.Metadata) error {
		t.Logf("Metadata: %+v; %+v", md.Header, md.Payload)
		metas[md.Header.MetaDataID] = md
		return nil
	}

	stream.StackBlockHandler = func(sb *nettrace.StackBlock) error {
		t.Logf("StackBlock %d; stacks: %d", sb.FirstID, len(sb.Stacks))
		for _, stack := range sb.Stacks {
			stacks[stack.ID] = stack.Data
		}
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
