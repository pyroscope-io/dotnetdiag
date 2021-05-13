package sampler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
)

type CPUTimeSampler struct {
	trace *nettrace.Trace
	sym   *symbols

	md      map[int32]*nettrace.Metadata
	stacks  map[int32][]uint64
	threads map[int64]*thread
}

type FrameInfo struct {
	ThreadID    int64
	Level       int
	SampledTime time.Duration
	Addr        uint64
	Name        string
}

// clrThreadSampleTraceData describes ThreadSample event payload for
// Microsoft-DotNETCore-SampleProfiler Event ID 0.
type clrThreadSampleTraceData struct {
	Type clrThreadSampleType
}

type clrThreadSampleType int32

const (
	_ clrThreadSampleType = iota - 1

	sampleTypeError
	sampleTypeExternal
	sampleTypeManaged
)

func NewCPUTimeSampler(trace *nettrace.Trace) *CPUTimeSampler {
	return &CPUTimeSampler{
		trace:   trace,
		sym:     newSymbols(),
		md:      make(map[int32]*nettrace.Metadata),
		threads: make(map[int64]*thread),
	}
}

func (s *CPUTimeSampler) Walk(fn func(FrameInfo)) {
	for tid := range s.threads {
		s.WalkThread(tid, fn)
	}
}

func (s *CPUTimeSampler) WalkThread(threadID int64, fn func(FrameInfo)) {
	t, ok := s.threads[threadID]
	if !ok {
		return
	}
	t.walk(func(i int, n *frame) {
		addr, _ := s.sym.resolveAddress(n.addr)
		name, _ := s.sym.resolveString(addr)
		fn(FrameInfo{
			ThreadID:    threadID,
			Addr:        addr,
			Name:        name,
			SampledTime: time.Duration(n.sampledTime),
			Level:       i,
		})
	})
}

func (s *CPUTimeSampler) EventHandler(e *nettrace.Blob) error {
	md, ok := s.md[e.Header.MetadataID]
	if !ok {
		return fmt.Errorf("metadata not found")
	}

	switch {
	case md.Header.ProviderName == "Microsoft-DotNETCore-SampleProfiler" && md.Header.EventID == 0:
		return s.addSample(e)

	case md.Header.ProviderName == "Microsoft-Windows-DotNETRuntimeRundown":
		switch {
		case md.Header.EventID == 144:
			return s.sym.addMethod(e)

		case md.Header.EventID == 152:
			return s.sym.addModule(e)
		}
	}

	return nil
}

func (s *CPUTimeSampler) MetadataHandler(md *nettrace.Metadata) error {
	s.md[md.Header.MetaDataID] = md
	return nil
}

func (s *CPUTimeSampler) StackBlockHandler(sb *nettrace.StackBlock) error {
	for _, stack := range sb.Stacks {
		s.addStack(stack)
	}
	return nil
}

func (s *CPUTimeSampler) SequencePointBlockHandler(*nettrace.SequencePointBlock) error {
	s.stacks = nil
	return nil
}

func (s *CPUTimeSampler) addStack(x nettrace.Stack) {
	if s.stacks == nil {
		s.stacks = make(map[int32][]uint64)
	}
	if s.trace.PointerSize == 8 {
		s.stacks[x.ID] = x.InstructionPointers64()
		return
	}
	s.stacks[x.ID] = x.InstructionPointers32()
	return
}

func (s *CPUTimeSampler) addSample(e *nettrace.Blob) error {
	d, err := parseClrThreadSampleTraceData(e.Payload)
	if err != nil {
		return err
	}
	// Relative time from the session start in milliseconds.
	relativeTime := e.Header.TimeStamp - s.trace.SyncTimeQPC
	s.thread(e.Header.ThreadID).addSample(d.Type, relativeTime, s.stacks[e.Header.StackID])
	return nil
}

func parseClrThreadSampleTraceData(b *bytes.Buffer) (clrThreadSampleTraceData, error) {
	var d clrThreadSampleTraceData
	err := binary.Read(b, binary.LittleEndian, &d)
	return d, err
}

func (s *CPUTimeSampler) thread(tid int64) *thread {
	t, ok := s.threads[tid]
	if ok {
		return t
	}
	t = &thread{callStack: new(callStack)}
	s.threads[tid] = t
	return t
}
