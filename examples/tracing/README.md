# Collect Tracing

The example demonstrates how the package may be used to collect and process `NetTrace` stream data using Diagnostics IPC:
the program processes events produced with **Microsoft-DotNETCore-SampleProfiler** provider and creates a sampled profile.

1. Run dotnet application.
2. Find its PID, e.g.:
   ```
   # dotnet-trace ps
   ```

3. Build and run the example program:
   ```
   # go run ./examples/tracing -p {pid}
   ```
