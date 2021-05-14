package sampler

import (
	"bytes"
	"container/heap"
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

	samples samples
}

type FrameInfo struct {
	ThreadID    int64
	Level       int
	SampledTime time.Duration
	Addr        uint64
	Name        string
}

type sample struct {
	typ          clrThreadSampleType
	threadID     int64
	stackID      int32
	timestamp    int64
	relativeTime int64
}

type samples []*sample

func (s samples) Len() int { return len(s) }

func (s samples) Less(i, j int) bool { return s[i].timestamp < s[j].timestamp }

func (s samples) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *samples) Push(x interface{}) { *s = append(*s, x.(*sample)) }

func (s *samples) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
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
		stacks:  make(map[int32][]uint64),
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
			Addr:        n.addr,
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
	for s.samples.Len() != 0 {
		x := heap.Pop(&s.samples).(*sample)
		s.thread(x.threadID).addSample(x.typ, x.relativeTime, s.stacks[x.stackID])
	}
	s.stacks = make(map[int32][]uint64)
	return nil
}

func (s *CPUTimeSampler) addStack(x nettrace.Stack) {
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
	heap.Push(&s.samples, &sample{
		typ:          d.Type,
		threadID:     e.Header.ThreadID,
		stackID:      e.Header.StackID,
		timestamp:    e.Header.TimeStamp,
		relativeTime: e.Header.TimeStamp - s.trace.SyncTimeQPC,
	})
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
	t = &thread{callTree: new(callTree)}
	s.threads[tid] = t
	return t
}
