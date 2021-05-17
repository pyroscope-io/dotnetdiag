# .Net diagnostics

The package provides means for .Net runtime diagnostics implemented in Golang:
 - [Diagnostics IPC Protocol](https://github.com/dotnet/diagnostics/blob/main/documentation/design-docs/ipc-protocol.md#transport) client.
 - [NetTrace](https://github.com/microsoft/perfview/blob/main/src/TraceEvent/EventPipe/EventPipeFormat.md) decoder.

### Diagnostic IPC Client

```
# go get github.com/pyroscope-io/dotnetdiag
```

Supported .Net versions:
 - .Net 5.0
 - .Net Core 3.1

Supported platforms:
 - [ ] Windows
 - [x] Linux
 - [x] MacOS

Implemented commands:
 - [x] StopTracing
 - [x] CollectTracing
 - [ ] CollectTracing2
 - [ ] CreateCoreDump
 - [ ] AttachProfiler
 - [ ] ProcessInfo
 - [ ] ResumeRuntime

### NetTrace decoder

```
# go get github.com/pyroscope-io/dotnetdiag/nettrace
```

Supported format versions: <= 4

The decoder deserializes `NetTrace` binary stream to the object sequence. The package also provides an example stream
handler implementation which processes **Microsoft-DotNETCore-SampleProfiler**
events: [github.com/pyroscope-io/dotnetdiag/nettrace/profiler](github.com/pyroscope-io/dotnetdiag/nettrace/profiler).

See [examples](examples) directory.

