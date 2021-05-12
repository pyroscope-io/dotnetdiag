package sampler

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
)

type Sampler struct {
	trace   *nettrace.Trace
	sym     *symbols
	md      map[int32]*nettrace.Metadata
	samples []map[int32]*Sample
}

type Sample struct {
	Stack nettrace.Stack
	Count uint64
}

// clrThreadSampleTraceData describes event payload fo
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

func NewSampler(trace *nettrace.Trace) *Sampler {
	return &Sampler{
		trace: trace,
		sym:   newSymbols(),
		md:    make(map[int32]*nettrace.Metadata),
	}
}

func (s *Sampler) EventHandler(e *nettrace.Blob) error {
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

func (s *Sampler) MetadataHandler(md *nettrace.Metadata) error {
	s.md[md.Header.MetaDataID] = md
	return nil
}

func (s *Sampler) StackBlockHandler(sb *nettrace.StackBlock) error {
	for _, stack := range sb.Stacks {
		s.append(stack)
	}
	return nil
}

func (s *Sampler) SequencePointBlockHandler(*nettrace.SequencePointBlock) error {
	s.samples = append(s.samples, make(map[int32]*Sample))
	return nil
}

func (s *Sampler) append(x nettrace.Stack) {
	if len(s.samples) == 0 {
		s.samples = []map[int32]*Sample{{x.ID: {Stack: x}}}
		return
	}
	ls := s.samples[len(s.samples)-1]
	smpl, ok := ls[x.ID]
	if ok {
		smpl.Stack = x
		ls[x.ID] = smpl
		return
	}
	ls[x.ID] = &Sample{Stack: x}
	return
}

func (s *Sampler) addSample(e *nettrace.Blob) error {
	d, err := parseClrThreadSampleTraceData(e.Payload)
	if err != nil {
		return err
	}
	if d.Type == sampleTypeError {
		return nil
	}
	if len(s.samples) == 0 {
		s.samples = []map[int32]*Sample{{e.Header.StackID: {Count: 1}}}
		return nil
	}
	ls := s.samples[len(s.samples)-1]
	if _, ok := ls[e.Header.StackID]; ok {
		ls[e.Header.StackID].Count++
	}
	return nil
}

func parseClrThreadSampleTraceData(b *bytes.Buffer) (clrThreadSampleTraceData, error) {
	var d clrThreadSampleTraceData
	err := binary.Read(b, binary.LittleEndian, &d)
	return d, err
}
