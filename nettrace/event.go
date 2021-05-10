package nettrace

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// MethodLoadUnloadTraceData describes event payload for
// Microsoft-Windows-DotNETRuntimeRundown Event ID 144. Based on:
// MethodLoadUnloadTraceDataBase, MethodLoadUnloadVerboseTraceData and
// ClrTraceEventParser.
type MethodLoadUnloadTraceData struct {
	MethodID           int64
	ModuleID           int64
	MethodStartAddress uint64
	MethodSize         int32
	MethodToken        int32
	MethodFlags        int32
	MethodNamespace    string
	MethodName         string
	MethodSignature    string
	//	ClrInstanceID      int16
	//	ReJITID            int64
}

func ParseMethodLoadUnloadTraceData(b *bytes.Buffer) (MethodLoadUnloadTraceData, error) {
	p := &parser{Buffer: b}
	var d MethodLoadUnloadTraceData
	p.read(&d.MethodID)
	p.read(&d.ModuleID)
	// MethodStartAddress, MethodSize, MethodToken, MethodFlags
	// may be omitted.
	p.read(&d.MethodStartAddress)
	p.read(&d.MethodSize)
	p.read(&d.MethodToken)
	p.read(&d.MethodFlags)
	d.MethodNamespace = p.utf16nts()
	d.MethodName = p.utf16nts()
	d.MethodSignature = p.utf16nts()
	return d, p.error()
}

func (d MethodLoadUnloadTraceData) String() string {
	// https://github.com/microsoft/perfview/blob/b087722375f5f694978c657d80309889a92bf7db/src/TraceEvent/Computers/TraceManagedProcess.cs#L4108
	// Output: <module_name>!System.Runtime.InteropServices.MemoryMarshal.AsMemory (value class System.ReadOnlyMemory`1<!!0>)
	p := strings.Index(d.MethodSignature, "(")
	if p < 0 {
		p = 0
	}
	return fmt.Sprintf("%s.%s%s", d.MethodNamespace, d.MethodName, d.MethodSignature[p:])
}

// ClrThreadSampleTraceData describes event payload fo
// Microsoft-DotNETCore-SampleProfiler Event ID 0.
type ClrThreadSampleTraceData struct {
	Type ClrThreadSampleType
}

type ClrThreadSampleType int32

const (
	_ ClrThreadSampleType = iota

	SampleTypeError
	SampleTypeExternal
	SampleTypeManaged
)

func ParseClrThreadSampleTraceData(b *bytes.Buffer) (ClrThreadSampleTraceData, error) {
	var d ClrThreadSampleTraceData
	err := binary.Read(b, binary.LittleEndian, &d)
	return d, err
}
