package sampler

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pyroscope-io/dotnetdiag/nettrace"
)

type symbols struct {
	addresses map[uint64]string
	methods   methods
	modules   map[int64]module
	sorted    bool
}

// method describes MethodLoadUnloadTraceData event payload for
// Microsoft-Windows-DotNETRuntimeRundown Event ID 144.
type method struct {
	MethodID           int64
	ModuleID           int64
	MethodStartAddress uint64
	MethodSize         int32
	MethodToken        int32
	MethodFlags        int32
	MethodNamespace    string
	MethodName         string
	MethodSignature    string
}

func (d method) String() string {
	p := strings.Index(d.MethodSignature, "(")
	if p < 0 {
		p = 0
	}
	return fmt.Sprintf("%s.%s%s", d.MethodNamespace, d.MethodName, d.MethodSignature[p:])
}

// module describes ModuleLoadUnloadTraceData event payload for
// Microsoft-Windows-DotNETRuntimeRundown Event ID 152.
type module struct {
	ModuleID     int64
	AssemblyID   int64
	ModuleFlags  int32
	ModuleILPath string
}

func (d module) String() string {
	return strings.TrimSuffix(filepath.Base(d.ModuleILPath), filepath.Ext(d.ModuleILPath))
}

type methods []method

func (m methods) Len() int { return len(m) }

func (m methods) Less(i, j int) bool { return m[i].MethodStartAddress < m[j].MethodStartAddress }

func (m methods) Swap(i, j int) { m[i], m[j] = m[j], m[i] }

func newSymbols() *symbols {
	return &symbols{
		addresses: make(map[uint64]string),
		modules:   make(map[int64]module),
	}
}

func (s *symbols) resolve(addr uint64) (string, bool) {
	if name, ok := s.addresses[addr]; ok {
		return name, true
	}
	if !s.sorted {
		sort.Sort(s.methods)
		s.sorted = true
	}
	methodIdx := sort.Search(len(s.methods), func(i int) bool {
		return s.methods[i].MethodStartAddress > addr
	})
	// Method not found.
	if methodIdx == len(s.methods) || methodIdx == 0 {
		return "?!?", false
	}
	met := s.methods[methodIdx-1]
	// Ensure the instruction pointer is within the method address space.
	if (met.MethodStartAddress + uint64(met.MethodSize)) <= addr {
		return "?!?", false
	}
	var name string
	mod, ok := s.modules[met.ModuleID]
	if !ok {
		name = fmt.Sprintf("?!%s", met)
	} else {
		name = fmt.Sprintf("%s!%s", mod, met)
	}
	s.addresses[addr] = name
	return name, true
}

func (s *symbols) addModule(e *nettrace.Blob) error {
	m, err := parseModule(e.Payload)
	if err != nil {
		return err
	}
	s.modules[m.ModuleID] = m
	return nil
}

func (s *symbols) addMethod(e *nettrace.Blob) error {
	m, err := parseMethod(e.Payload)
	if err != nil {
		return err
	}
	s.methods = append(s.methods, m)
	s.sorted = false
	return nil
}

func parseMethod(b *bytes.Buffer) (method, error) {
	p := &nettrace.Parser{Buffer: b}
	var d method
	p.Read(&d.MethodID)
	p.Read(&d.ModuleID)
	p.Read(&d.MethodStartAddress)
	p.Read(&d.MethodSize)
	p.Read(&d.MethodToken)
	p.Read(&d.MethodFlags)
	d.MethodNamespace = p.UTF16NTS()
	d.MethodName = p.UTF16NTS()
	d.MethodSignature = p.UTF16NTS()
	return d, p.Err()
}

func parseModule(b *bytes.Buffer) (module, error) {
	p := &nettrace.Parser{Buffer: b}
	var d module
	p.Read(&d.ModuleID)
	p.Read(&d.AssemblyID)
	p.Read(&d.ModuleFlags)
	_ = p.Next(12)
	d.ModuleILPath = p.UTF16NTS()
	return d, p.Err()
}
