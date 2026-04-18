---
name: dotnet-worker-windows-service
description: .NET Worker Service on Windows registered as a Windows Service; detect by executable name matching project namespace (e.g. App.WindowsService.exe), presence of Microsoft.Extensions.Hosting.WindowsServices in memory/modules, or service registry entry with binary path pointing to published .exe in a custom directory.
---

# .NET Worker Service on Windows — Windows Service

## Overview

A .NET Worker Service is a long-running background application built on the .NET generic host and BackgroundService framework. When deployed as a Windows Service, it runs under the Windows Service Control Manager (SCM) and is registered in the Windows services database. This variant is common in enterprise environments where legacy Windows Service infrastructure is required; teams choose it for background job processing, scheduled tasks, and data integration without a web UI. Deployments typically use .NET 8.0 or later with the `Microsoft.NET.Sdk.Worker` project SDK.

## Deployment Process

1. **Create the worker project:**
   ```
   dotnet new worker --name MyWorkerService
   cd MyWorkerService
   ```

2. **Install the Windows Services NuGet package:**
   ```
   dotnet package add Microsoft.Extensions.Hosting.WindowsServices
   ```

3. **Update Program.cs to add Windows Service support:**
   Call `builder.Services.AddWindowsService(options => { options.ServiceName = "My Service Name"; })` before `builder.Build()`.

4. **Update Worker class to extend BackgroundService and implement ExecuteAsync.**

5. **Publish the app as a single-file executable:**
   ```
   dotnet publish --configuration Release --output "C:\MyWorkerService" 
   ```
   Ensure the `.csproj` file includes:
   - `<PublishSingleFile>true</PublishSingleFile>`
   - `<RuntimeIdentifier>win-x64</RuntimeIdentifier>`

6. **Register the Windows Service using sc.exe with Administrator privileges:**
   ```
   sc.exe create "ServiceName" binpath= "C:\MyWorkerService\MyWorkerService.exe"
   ```

7. **Start the service:**
   ```
   sc.exe start "ServiceName"
   ```

8. **Verify in Services app or Event Viewer (Application logs).**

## Process Signatures

- **Process name / executable:** The executable name derives from the project root namespace (e.g., `App.WindowsService.exe`, `MyCompany.Worker.exe`). The executable is published as a single `.exe` file.
- **Command-line patterns:** When run by SCM, typically `C:\Path\To\ServiceName.exe` with no arguments (or optional `--contentRoot` argument). No console window is visible.
- **Parent process:** `services.exe` (Windows Service Host process, PID typically 680-750).
- **Typical user:** `SYSTEM`, `NT AUTHORITY\LOCAL SERVICE`, or a custom Windows user account (if configured via sc.exe failure/config commands).
- **Working directory:** The directory containing the `.exe` (e.g., `C:\MyWorkerService\`), or optionally set via `--contentRoot` argument in `sc.exe create` binpath.

## File System Paths

### Windows
- **Install root:** `C:\Program Files\<CompanyName>\<ServiceName>\` or `C:\<Custom>\<ServiceName>\` (operator-chosen).
- **Binaries:** Single `.exe` file at the install root (e.g., `C:\Program Files\MyCompany\MyWorker\MyWorker.exe`). No separate DLLs required when published as single-file. All dependencies are bundled.
- **Application/deploy dir:** Same as install root; this is where `appsettings.json`, `appsettings.Production.json`, and any configuration files reside.

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `DOTNET_ENVIRONMENT` | Sets the hosting environment (Development, Staging, Production) for config file selection and logging filter level. | `Production` |
| `ASPNETCORE_ENVIRONMENT` | Legacy variant; `DOTNET_ENVIRONMENT` is preferred in .NET Worker Services. | `Production` |

Additional variables (DOTNET_-prefixed) are loaded by the host configuration builder at startup from the process environment.

## Configuration Files

- **appsettings.json** — Application configuration in JSON format. Located in the working directory (same as the `.exe`). Contains logging levels, custom app settings, and dependency injection configuration. Mandatory when the app reads configuration via `IConfiguration`.
- **appsettings.{Environment}.json** — Environment-specific overrides (e.g., `appsettings.Production.json`). Loaded after `appsettings.json` if the environment matches.
- **hostsettings.json** — (Optional) Host-level configuration for lifetime, logging providers, and content root path. Rarely used in Windows Service deployments.

## Log Locations

- **Windows Event Log (Application):** Primary log destination for .NET Worker Services running as Windows Services. Logs are written via the `EventLogLoggerProvider`. Access via **Event Viewer > Windows Logs > Application**. Look for entries with **Source** matching the namespace or custom source name set in `EventLogSettings.SourceName` (e.g., `"The Joke Service"`).
- **Console output (none visible):** When running as a service, console output is suppressed (no console window). If the service is run manually (not via SCM), output goes to the command prompt.
- **File-based logging:** If the app adds a file logger in `Program.cs`, logs go to the configured file path (e.g., `C:\MyWorkerService\logs\worker.log`), but event log is the standard.

## Service / Init Integration

The service is registered with the **Windows Service Control Manager (SC.exe)** and appears in **Services.msc**. 

- **Service name in registry:** `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>` (e.g., `Services\.NET Joke Service`).
- **Binary path stored in registry:** `ImagePath` value under the service key (e.g., `C:\MyWorkerService\MyWorkerService.exe` or with `/` flags for alternate start path).
- **Service type:** `WIN32_OWN_PROCESS` (type 10 in sc.exe output).
- **Startup type:** Configurable via `sc.exe config` (default is `auto` after creation). Common values: `auto`, `demand`, `disabled`.
- **Recovery actions:** Configurable via `sc.exe failure` command (e.g., restart on failure after 60 seconds).

## Detection Heuristics

1. **Executable path:** Look for a `.exe` file in a custom directory (not `C:\Windows\System32`). If named `App.WindowsService.exe`, `MyWorker.exe`, or similar with `.NET`-like assembly naming, it is likely a .NET Worker Service.

2. **Process parent:** If parent is `services.exe`, the process is registered as a Windows Service.

3. **Loaded modules:** Check `GetModuleHandle` or scan the process memory maps (via WMI or Toolhelp) for:
   - `Microsoft.Extensions.Hosting.dll`
   - `Microsoft.Extensions.Hosting.WindowsServices.dll`
   - `System.Runtime.dll` (all .NET apps, but combined with above is strong signal)

4. **Windows Event Log source name:** If a custom `EventLogSettings.SourceName` is configured (e.g., "The Joke Service"), that source will appear in the Application log when the service logs. Cross-reference source name with running process to confirm.

5. **Registry service definition:** Query `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>\ImagePath`. If the path ends in `.exe` and the parent process is `services.exe`, it is a Windows Service. If the exe name matches the running process, it is the service executable.

6. **Command-line arguments (rarely present):** Most Windows Services are called with no arguments or minimal flags (e.g., `--contentRoot C:\...`). No `-jar`, `-main`, or script interpreter in the command line.

7. **User account:** If running under `NT AUTHORITY\LOCAL SERVICE` or `NT AUTHORITY\SYSTEM` (not a standard user), it indicates a service or system process.

## Version / Variant Differences

- **.NET 8.0 vs .NET 9.0 vs .NET 10.0:** The generic host and BackgroundService APIs are stable across these versions. Core detection logic remains the same. Newer versions may include additional logging providers or lifetime APIs, but the Windows Service registration and execution model is unchanged.
- **Single-file vs. multi-file publishing:** Most production deployments use single-file (`.exe` + bundled dependencies). Multi-file deployments (`.exe` + separate DLLs) are less common but possible; detection should not assume single-file.
- **Self-contained vs. framework-dependent:** Single-file is typically self-contained (no separate .NET runtime required). Framework-dependent deployments are rare in Windows Service contexts but do not change detection signals.
- **.NET Joke Service example (from Microsoft docs):** The reference implementation uses `options.ServiceName = ".NET Joke Service"` and logs to the Event Log. This is a representative example; real-world service names and logging strategies vary.

## Sources
- [Create Windows Service using BackgroundService - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/windows-service)
- [Worker Services - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/workers)
- [.NET Generic Host - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/generic-host)
