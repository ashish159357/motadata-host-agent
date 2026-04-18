---
name: dotnet-worker-windows-service
description: .NET Worker Service on Windows registered as a Windows Service via UseWindowsService() or sc.exe; detectable by dotnet.exe parent process running as SYSTEM or service account, presence of Microsoft.Extensions.Hosting.WindowsServices in assembly, or Windows Service registry entries with binPath pointing to .NET executable.
---

# .NET Worker Service on Windows — Windows Service Registration

## Overview

A .NET Worker Service deployed as a Windows Service is a long-running background process that uses the .NET generic host and `BackgroundService` abstraction, registered with Windows Service Control Manager (SCM) via `sc.exe` or installer. This is the standard pattern for deploying .NET-based services on Windows machines without containers. Teams choose this variant for CPU-intensive background processing, queuing, scheduled operations, and daemon-like workloads that require automatic restart on failure and integration with Windows event logs.

## Deployment Process

1. **Create or scaffold a Worker Service project:**
   ```
   dotnet new worker --name MyWorkerService
   ```

2. **Install the Windows Service NuGet package:**
   ```
   dotnet package add Microsoft.Extensions.Hosting.WindowsServices
   ```
   (Or older style: `dotnet add package Microsoft.Extensions.Hosting.WindowsServices`)

3. **Configure the host in Program.cs** to call `AddWindowsService()` or `UseWindowsService()`:
   ```csharp
   var builder = Host.CreateApplicationBuilder(args);
   builder.Services.AddWindowsService(options =>
   {
       options.ServiceName = "MyWorkerService";
   });
   builder.Services.AddHostedService<Worker>();
   var host = builder.Build();
   host.Run();
   ```

4. **Publish as a self-contained executable** (recommended):
   ```
   dotnet publish -c Release -r win-x64 --self-contained -o ./publish
   ```

5. **Register the service with Windows SCM** using `sc.exe` (run as Administrator):
   ```
   sc.exe create "MyWorkerService" binpath= "C:\Path\To\MyWorkerService.exe"
   ```
   (Note: space after `binpath=` is required; quoted paths with spaces must be double-quoted.)

6. **(Optional) Configure recovery behavior:**
   ```
   sc.exe failure "MyWorkerService" reset= 0 actions= restart/60000/restart/60000/run/1000
   ```

7. **Start the service:**
   ```
   sc.exe start "MyWorkerService"
   ```

## Process Signatures

- **Process name / executable:** `dotnet.exe` (if running as a DLL via `dotnet MyWorkerService.dll`) or the published `.exe` file (e.g., `MyWorkerService.exe`)
- **Command-line patterns:** 
  - `C:\Path\To\MyWorkerService.exe` (direct invocation for single-file publish)
  - `dotnet.exe C:\Path\To\MyWorkerService.dll` (running via .NET runtime)
  - May include `--contentRoot` argument specifying application directory
- **Parent process:** `services.exe` (Windows Service Control Manager) or `svchost.exe` when hosted in a svchost process (rare for custom services)
- **Typical user:** `SYSTEM`, `LOCAL SERVICE`, or a service account (e.g., `NT SERVICE\MyWorkerService`)
- **Working directory:** Application directory containing the executable or DLL (e.g., `C:\Path\To\MyWorkerService\`)

## File System Paths

### Windows

- **Install root:** Operator-defined; typically `C:\Program Files\MyWorkerService\`, `C:\Services\MyWorkerService\`, or `C:\opt\MyWorkerService\`
- **Binaries:** 
  - Single-file executable: `C:\[InstallRoot]\MyWorkerService.exe`
  - DLL-based: `C:\[InstallRoot]\MyWorkerService.dll` (requires `dotnet.exe` in PATH or explicit binpath)
  - Dependencies (for non-self-contained): .NET runtime DLLs in same directory or system PATH
- **Application/deploy dir:** `C:\[InstallRoot]\` (all application files, config, logs)
- **Configuration files:** `appsettings.json`, `appsettings.{Environment}.json` in application directory
- **Log locations:** 
  - Windows Event Log: **Application** log, source name set in `Program.cs` (e.g., `EventLogLoggerProvider` with `SourceName` = app namespace)
  - File logs (if configured): operator-defined, typically `C:\[InstallRoot]\logs\` or `C:\ProgramData\MyWorkerService\logs\`

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `DOTNET_ROOT` | Path to .NET runtime (if DLL-based and runtime not in PATH) | `C:\Program Files\dotnet` |
| `ASPNETCORE_ENVIRONMENT` | Selects configuration file variant for generic host (e.g., `appsettings.Production.json`) | `Production`, `Development`, `Staging` |
| `DOTNET_MULTILEVEL_LOOKUP` | Controls whether .NET probes for runtime in user profile (set to 0 to disable) | `0` or `1` (default) |
| `DOTNET_TieredCompilation` | Enable JIT tiering (code generation optimization) | `1` (default, enabled) |

Note: Custom environment variables passed during service registration via `sc.exe` parameters are not automatically inherited; use `appsettings.json` for configuration instead.

## Configuration Files

- **appsettings.json** — Primary configuration for generic host and logging, located in application directory (e.g., `C:\Program Files\MyWorkerService\appsettings.json`)
- **appsettings.{Environment}.json** — Environment-specific overrides, where `{Environment}` matches the `ASPNETCORE_ENVIRONMENT` variable (e.g., `appsettings.Production.json`)
- **Project file (.csproj)** — Not deployed; defines build properties like `PublishSingleFile`, `RuntimeIdentifier`, `TargetFramework`, `OutputType`
- **.NET configuration/runtime config** — Optional `runtimeconfig.json` and `.deps.json` files (auto-generated for DLL-based deployments; embedded in single-file executable)

## Log Locations

- **Windows Event Log:** Logs from `ILogger` configured with `EventLogLoggerProvider` appear in **Event Viewer** > **Windows Logs** > **Application** with the source name specified in `appsettings.json` (e.g., `"EventLog": { "SourceName": "MyWorkerService" }`)
- **File-based logs (if configured):** Operator-defined location; recommended pattern: `C:\ProgramData\MyWorkerService\logs\app-{date}.log`
- **No built-in stdout/stderr redirection:** Service output is not written to files by default unless explicitly configured in code or config; console output is lost.

## Service / Init Integration

**Windows Service Control Manager (SCM):**

- **Service name:** Operator-defined during `sc.exe create` registration (e.g., `MyWorkerService`)
- **Registry location:** `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\[ServiceName]`
- **Key registry values:**
  - `ImagePath`: Full path to executable or `dotnet.exe` with DLL path (e.g., `C:\Program Files\dotnet\dotnet.exe C:\Services\MyWorkerService.dll`)
  - `ObjectName`: User account the service runs under (default: `LocalSystem`)
  - `Start`: Start type (2 = automatic, 3 = manual, 4 = disabled)
- **Recovery policy:** Configured via `sc.exe failure` command; enables automatic restart on exit code `1` (non-zero exit from `BackgroundService`)
- **Service lifecycle:**
  - Startup: `sc.exe start [ServiceName]` → SCM launches process
  - Graceful shutdown: `sc.exe stop [ServiceName]` → SCM signals `CancellationToken` to hosted service
  - Abnormal exit: Monitored by SCM; recovery policy determines restart behavior

**Services Management Console:** Viewable and manageable in `services.msc` (Windows Services GUI).

## Detection Heuristics

The following unambiguous signals identify a .NET Worker Service registered as a Windows Service:

1. **Process ancestry:** Parent process is `services.exe` (Windows SCM)
2. **Executable path:** Ends with `.exe` and located in a custom application directory (not `C:\Windows\System32\`)
3. **User account:** Running as `SYSTEM`, `LOCAL SERVICE`, or `NT SERVICE\[ServiceName]` (not a regular user)
4. **Registry entry:** `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\[ServiceName]` exists with `ImagePath` pointing to the .NET executable
5. **Assembly manifest:** Presence of `Microsoft.Extensions.Hosting.WindowsServices` in loaded assemblies (via `.deps.json` or binary inspection)
6. **Logging configuration:** `EventLogLoggerProvider` configured in `appsettings.json` with non-null `SourceName` indicates Windows Service integration
7. **Command-line absence:** No `/start`, `/install`, or `--worker` arguments; service parameters defined entirely in `Program.cs` and config files
8. **Exit behavior:** Process exits cleanly on shutdown signal (no hanging processes); recovery policy in SCM registry dictates restart

## Version / Variant Differences

- **.NET Core 3.0 – 3.1:** `UseWindowsService()` extension available via `Microsoft.Extensions.Hosting.WindowsServices`; simple integration with `IHost`
- **.NET 5.0 – 9.0:** Same pattern; `UseWindowsService()` remains primary method; `AddWindowsService()` introduced in .NET 6+ as alternative (more modern builder API)
- **.NET 10.0+:** Builder-pattern config (`AddWindowsService()` on `HostApplicationBuilder`) preferred; `UseWindowsService()` still supported for backward compatibility
- **Publishing changes:** .NET 5+ emphasizes single-file executables (`PublishSingleFile=true`); DLL-based deployments still valid but less common
- **IHostedService exception handling:** .NET 6+ changed default behavior of `BackgroundServiceExceptionBehavior` from `Ignore` (zombie services) to `StopHost` (clean exit); code must call `Environment.Exit(1)` on exception to enable SCM recovery policy
- **Configuration:** All variants use `appsettings.json` + generic host; no breaking differences in config format

## Sources

- [Create Windows Service using BackgroundService - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/windows-service)
- [Worker Services - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/workers)
- [Running a .NET Core Generic Host App as a Windows Service - Steve Gordon](https://www.stevejgordon.co.uk/running-net-core-generic-host-applications-as-a-windows-service)
- [.NET Generic Host - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/generic-host)
