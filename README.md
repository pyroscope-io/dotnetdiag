# .Net diagnostics

*Work In Progress*

The package provides means for .Net runtime diagnostics implemented in Golang:
 - [Diagnostics IPC Protocol](https://github.com/dotnet/diagnostics/blob/main/documentation/design-docs/ipc-protocol.md#transport) client.
 - [NetTrace](https://github.com/microsoft/perfview/blob/main/src/TraceEvent/EventPipe/EventPipeFormat.md) decoder.

### Diagnostics IPC Protocol Client

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

See [examples](examples) directory. 
