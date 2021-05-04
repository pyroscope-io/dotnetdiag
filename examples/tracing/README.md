# Collect Tracing

The example demonstrates how the package may be used to collect NetTrace stream data using Diagnostics IPC:
the program streams traces produced with `Microsoft-DotNETCore-SampleProfiler` provider to a file.

1. Run dotnet application.
2. Find its PID, e.g.:
   ```
   # dotnet-trace ps
   ```

3. Find Diagnostics IPC socket/pipe created by the application, e.g.:
    - Linux/MacOS:
      ```
      # lsof -p {PID} | grep dotnet-diagnostic
      ```

4. Run the example program:
   ```
   # collect -s {absolute-path-to-socket} -o {path-to-nettrace-output-file}
   ```   

5. (Optional) Verify output file:
   ```
   # dotnet-trace convert --format speedscope -o {path-to-output-file} {path-to-nettrace-file}
   ```
