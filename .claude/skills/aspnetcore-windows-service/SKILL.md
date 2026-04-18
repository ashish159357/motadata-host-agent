---
name: aspnetcore-windows-service
description: ASP.NET Core web application registered as a Windows Service via sc.exe; detectable by dotnet.exe or *.exe child process, ASPNETCORE_* environment variables, appsettings.json in working directory, and libcoreclr.so or Windows Service entry in registry.
---

# ASP.NET Core on Windows — Windows Service

## Overview

ASP.NET Core applications can be hosted as Windows Services without IIS, allowing them to start automatically on server reboots and run under a service identity. This deployment pattern is common in enterprise environments where background tasks or long-running web APIs need lifecycle management without a web server dependency. Teams choose this variant when they need application auto-restart capabilities and want to avoid IIS overhead.

## Deployment Process

1. **Create a Worker Service or ASP.NET Core project** (Visual Studio template or `dotnet new worker`)
2. **Add the NuGet package** `Microsoft.Extensions.Hosting.WindowsServices`
3. **Call `AddWindowsService()`** in `Program.cs` (e.g., `builder.Services.AddWindowsService();`)
4. **Publish as a self-contained or framework-dependent executable** using `dotnet publish -c Release -r win-x64 --self-contained` (or similar)
5. **Register the Windows Service** using `sc.exe create` as Administrator:
   ```powershell
   sc.exe create "ServiceName" binpath= "C:\Path\To\App.exe"
   ```
6. **(Optional) Configure recovery options** using `sc.exe failure`
7. **Start the service** using `sc.exe start "ServiceName"`

## Process Signatures

- **Process name / executable:** `dotnet.exe` (if framework-dependent) or the published `.exe` name (e.g., `MyApp.exe`)
- **Command-line patterns:** `dotnet.exe C:\Path\To\App.dll` or direct invocation of the published `.exe`; may include `--contentRoot` argument
- **Parent process:** `services.exe` (Windows Service Control Manager) or `svchost.exe` in some configurations
- **Typical user:** `SYSTEM`, `LOCAL SERVICE`, `NETWORK SERVICE`, or a configured custom domain account
- **Working directory:** By default, `C:\Windows\System32` (must explicitly configure via `--contentRoot` or `IHostBuilder.UseContentRoot()` to use a different path)

## File System Paths

### Windows

- **Install root:** Typically `C:\Program Files\AppName\` or `C:\Program Files (x86)\AppName\` (or custom location specified at deployment)
- **Binaries:** `C:\Program Files\AppName\App.exe` (self-contained) or `C:\Program Files\AppName\App.dll` (framework-dependent, with `dotnet.exe` in DOTNET_ROOT)
- **Application/deploy dir:** `C:\Program Files\AppName\` (contains executable, DLLs, and supporting files)
- **Configuration:** `C:\Program Files\AppName\appsettings.json`, `C:\Program Files\AppName\appsettings.{Environment}.json`
- **Logs:** Windows Event Log (Application log, source name = application namespace or configured event source)

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `ASPNETCORE_ENVIRONMENT` | Determines which `appsettings.{Environment}.json` is loaded | `Production`, `Development`, `Staging` |
| `ASPNETCORE_URLS` | Bind address(es) for the HTTP server | `http://localhost:5000` or `http://+:80` |
| `ASPNETCORE_HTTPS_PORT` | HTTPS port if applicable | `443` |
| `DOTNET_ENVIRONMENT` | Alternative environment variable (older naming) | `Production`, `Development` |
| `DOTNET_RUNNING_IN_CONTAINER` | Set by framework; indicates container mode (false for Windows Service) | `false` |

## Configuration Files

- **appsettings.json** — Main application configuration; located in the published directory (e.g., `C:\Program Files\AppName\appsettings.json`)
- **appsettings.{Environment}.json** — Environment-specific overrides (e.g., `appsettings.Production.json` loaded when `ASPNETCORE_ENVIRONMENT=Production`)
- **appsettings.Development.json** — Developer-only settings (typically not deployed)
- **web.config** — (Optional, IIS-only; not used for Windows Service deployment; ignore if present)

## Log Locations

- **Windows Event Log (Primary):** Application log entries with source name matching the application namespace or explicit `EventLogSettings.SourceName` configured in `appsettings.json`
- **File-based logs:** If configured via ILogger or third-party provider (e.g., Serilog), typically written to `C:\Program Files\AppName\logs\` or a path specified in `appsettings.json`
- **Query Event Log:** Open Event Viewer → Windows Logs → Application, filter by the source name

## Service / Init Integration

- **Service Name:** Arbitrary name assigned by operator during `sc.exe create` command (e.g., `"MyWebService"`, `.NET Joke Service`)
- **Service Manager:** Windows Service Control Manager (`services.exe`)
- **Service Registry:** `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>` (binary path stored in `ImagePath` value)
- **Lifecycle commands:**
  - Start: `sc.exe start "ServiceName"`
  - Stop: `sc.exe stop "ServiceName"`
  - Delete: `sc.exe delete "ServiceName"`
  - Query: `sc.exe query "ServiceName"`
- **Auto-start:** Configured via `sc.exe config "ServiceName" start= auto` (default is `demand`)
- **Recovery behavior:** Configured via `sc.exe failure "ServiceName"` (e.g., automatic restart on crash)

## Detection Heuristics

1. **Highest confidence:** Process parent is `services.exe` AND executable is named `*.exe` AND process environment contains `ASPNETCORE_*` variable (e.g., `ASPNETCORE_ENVIRONMENT`, `ASPNETCORE_URLS`)
2. **High confidence:** Executable path is under `C:\Program Files\` AND `appsettings.json` exists in the same directory AND contains `"Logging"` or `"Kestrel"` JSON keys
3. **High confidence:** Command-line contains `dotnet.exe` with a `.dll` argument AND `ASPNETCORE_ENVIRONMENT` variable is set
4. **Medium confidence:** Process working directory is `C:\Windows\System32` AND process name is `dotnet.exe` OR custom `.exe` AND parent is `services.exe`
5. **Registry check:** Query `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\` for entries with `ImagePath` containing `.exe` or `dotnet.exe` followed by a `.dll` path

## Version / Variant Differences

- **ASP.NET Core 3.1–5.0:** `UseWindowsService()` extension available; Worker Service template recommended; logging defaults to Event Log
- **ASP.NET Core 6.0–10.0:** Same as 3.1–5.0; `AddWindowsService()` replaces `UseWindowsService()` in newer sample code; Event Log source creation restricted to Admin users
- **Framework-dependent vs. self-contained:** 
  - Framework-dependent: `dotnet.exe` invoked with `.dll` argument; requires .NET runtime installed
  - Self-contained: Published `.exe` runs standalone; includes runtime in binary; common in production
- **Content root:** By default set to `AppContext.BaseDirectory` (the executable's directory) when running as a Windows Service; operators may override via `--contentRoot` argument or `IHostBuilder.UseContentRoot()`
- **.NET 6 exception handling:** `BackgroundServiceExceptionBehavior.StopHost` is default; unhandled exceptions stop the service (older versions defaulted to `Ignore`, causing zombie processes)

## Sources

- [Host ASP.NET Core in a Windows Service | Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/windows-service?view=aspnetcore-10.0)
- [Create Windows Service using BackgroundService - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/windows-service)
- [sc.exe create command | Microsoft Learn](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/sc-create)
