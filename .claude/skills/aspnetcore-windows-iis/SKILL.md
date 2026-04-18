---
name: aspnetcore-windows-iis
description: ASP.NET Core on Windows IIS via ASP.NET Core Module; detectable by w3wp.exe or dotnet.exe processes, `web.config` with `aspNetCore` element, and `hostingModel` attribute (inprocess or outofprocess).
---

# ASP.NET Core on Windows IIS — In-Process / Out-of-Process Hosting

## Overview

ASP.NET Core applications deployed to Internet Information Services (IIS) on Windows Server use the ASP.NET Core Module (ANCM/AspNetCoreModuleV2) as a native IIS module to handle HTTP requests. The module supports two hosting models: in-process (runs app in the same process as the IIS worker) and out-of-process (runs app in a separate Kestrel process). In-process hosting has been the default since ASP.NET Core 3.0 and provides better performance by avoiding loopback network proxying. This is the most common production deployment pattern for ASP.NET Core on Windows enterprise infrastructure.

## Deployment Process

1. **Publish the application** from Visual Studio or command line:
   ```
   dotnet publish -c Release -o C:\inetpub\wwwroot\myapp
   ```

2. **Install ASP.NET Core Runtime** (if framework-dependent deployment):
   ```
   Download from https://dotnet.microsoft.com/download and install
   ```

3. **Create IIS Application Pool** (via IIS Manager or PowerShell):
   - Set pipeline mode to "Integrated"
   - Set .NET CLR version to "No Managed Code" (ANCM is native, not CLR-based)
   - Match bitness (x64 or x86) to the app's target architecture

4. **Create IIS Website/Application** (via IIS Manager):
   - Point physical path to the published app root (where `web.config` is located)
   - Bind to HTTP/HTTPS port (typically 80 or 443)
   - Assign to the application pool created in step 3

5. **Verify web.config** is present at the app root:
   - For framework-dependent: `<aspNetCore processPath="dotnet" arguments=".\MyApp.dll" hostingModel="inprocess" />`
   - For self-contained: `<aspNetCore processPath=".\MyApp.exe" hostingModel="inprocess" />`

6. **Enable required IIS features**:
   ```
   Windows Feature: IIS > Application Development Features > ASP.NET Core Hosting Bundle
   ```

7. **Start the application pool** and verify the site is running via HTTP.

## Process Signatures

- **Process name / executable:**
  - **In-process:** `w3wp.exe` (IIS worker process) — the ASP.NET Core app runs inside this process
  - **Out-of-process:** `dotnet.exe` or `MyApp.exe` (self-contained) — the app runs in a separate process; IIS worker (`w3wp.exe` or `iisexpress.exe`) acts as reverse proxy

- **Command-line patterns:**
  - In-process: `w3wp.exe -appPool "MyAppPool"`
  - Out-of-process: `dotnet.exe .\MyApp.dll` or `.\MyApp.exe` (launched by ANCM with environment variable `ASPNETCORE_PORT` set to a random high port, e.g., 12345)

- **Parent process:**
  - `services.exe` → `svchost.exe` (WAS — Windows Process Activation Service) → `w3wp.exe` or `iisexpress.exe`
  - For out-of-process, the child app process (dotnet.exe or MyApp.exe) is spawned by w3wp.exe

- **Typical user:**
  - `IUSR` (IIS anonymous identity) or `ApplicationPoolIdentity` (app pool-specific identity)
  - May be overridden to a custom domain user or service account for resource access

- **Working directory:**
  - The physical path of the IIS website (typically `C:\inetpub\wwwroot\myapp` or custom directory)
  - Configured in IIS site properties or `web.config` location element

## File System Paths

### Windows

- **Install root:**
  - ASP.NET Core Runtime: `C:\Program Files\dotnet\` (for framework-dependent deployments)
  - IIS: `C:\Windows\System32\inetsrv\`
  - ASP.NET Core Module: `C:\Windows\System32\inetsrv\aspnetcore.dll` (native module, loaded by IIS)

- **Binaries:**
  - Runtime binaries: `C:\Program Files\dotnet\dotnet.exe`, `C:\Program Files\dotnet\shared\Microsoft.NETCore.App\{version}\`
  - Self-contained app: `C:\inetpub\wwwroot\myapp\MyApp.exe` (and all dependencies in the same directory)

- **Application/deploy dir:**
  - Default: `C:\inetpub\wwwroot\{appname}\` (or any custom physical path configured in IIS)
  - Must contain: `web.config`, `*.dll` (app and framework assemblies), `appsettings.json`, `appsettings.{Environment}.json`, static files

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `ASPNETCORE_ENVIRONMENT` | Deployment environment (Development/Staging/Production) | `Production` (set in web.config or system) |
| `ASPNETCORE_URLS` | Server listening address(es) — in-process uses IIS binding, out-of-process listens on localhost with port from `ASPNETCORE_PORT` | `http://localhost:5000` (out-of-process); ignored in-process |
| `ASPNETCORE_PORT` | Port for out-of-process Kestrel to listen on; set by ANCM | random high port (e.g., `12345`) |
| `ASPNETCORE_CONTENTROOT` | Base path for the app (usually the web.config directory) | `C:\inetpub\wwwroot\myapp` |
| `ASPNETCORE_APPLICATIONBASE` | Application base path (set by IIS integration) | same as content root |
| `DOTNET_ROOT` | Root directory of the .NET runtime (out-of-process) | `C:\Program Files\dotnet` |
| `DOTNET_MULTILEVEL_LOOKUP` | Enable multi-level .NET installation lookup | `0` (usually disabled in IIS) |

## Configuration Files

- **web.config** — Located at the application root (`C:\inetpub\wwwroot\{appname}\web.config`). Contains ASP.NET Core Module configuration with `<aspNetCore>` element:
  - `processPath`: Path to `dotnet` or `MyApp.exe`
  - `arguments`: For framework-dependent, the .dll file name (e.g., `.\MyApp.dll`)
  - `hostingModel`: `inprocess` (default since 3.0) or `outofprocess`
  - `stdoutLogEnabled`: Boolean; if `true`, stdout/stderr logged to disk
  - `stdoutLogFile`: Path for stdout logs (relative to app root, e.g., `.\logs\stdout`); ANCM appends timestamp and `.log` extension
  - `rapidFailsPerMinute`: Max crashes per minute (out-of-process only; default 10)
  - `requestTimeout`: Timeout for requests to out-of-process app (default 2 minutes)
  - `startupTimeLimit`: Seconds to wait for process startup (default 120)
  - `shutdownTimeLimit`: Seconds to wait for graceful shutdown (default 10)
  - Environment variables can be set via `<environmentVariables>` child element

- **appsettings.json** — Standard ASP.NET Core configuration file at the app root; loaded at runtime

- **appsettings.{Environment}.json** — Environment-specific settings (e.g., `appsettings.Production.json`); loaded if `ASPNETCORE_ENVIRONMENT` matches

## Log Locations

- **ANCM debug logs** (for troubleshooting module issues):
  - Location: `C:\Windows\System32\LogFiles\HTTPERR\`
  - File naming: `httperr{date}.log`

- **Application stdout/stderr logs** (if `stdoutLogEnabled="true"` in web.config):
  - Base path: Value of `stdoutLogFile` attribute (relative to app root, default `.\logs\stdout`)
  - Example: `C:\inetpub\wwwroot\myapp\logs\stdout_20240418_123456_1234.log` (timestamp and PID added by ANCM)
  - Out-of-process Kestrel and application exceptions are captured here

- **Event Viewer logs** (Windows Event Viewer):
  - Application log entries for ASP.NET Core Module errors and crashes
  - Look for source "IIS AspNetCore Module" or application events

- **IIS logs** (if HTTP logging enabled):
  - Default: `C:\inetpub\logs\LogFiles\W3SVC{siteID}\`
  - Contains HTTP request/response details per IIS configuration

## Service / Init Integration

ASP.NET Core on IIS is managed by:

- **Windows Process Activation Service (WAS):** System service (`WAS` service in Services.msc) that manages IIS application pools and worker processes
- **W3SVC (World Wide Web Publishing Service):** IIS system service that manages websites and modules
- **Application Pool:** Logical container within IIS that runs `w3wp.exe` (or `iisexpress.exe` for IIS Express)

There is no systemd unit or Windows service wrapper for the app itself; IIS/WAS manage the lifecycle. Application pools can be configured to auto-start on server boot via IIS Manager:
- Right-click Application Pool → Advanced Settings → Start Mode: `AlwaysRunning`

For out-of-process deployments, the ANCM spawns the app process on first request and restarts it on failure (up to `rapidFailsPerMinute` limit).

## Detection Heuristics

The highest-confidence signals for detecting ASP.NET Core on IIS:

1. **web.config file at site root** with:
   - `<system.webServer>` element
   - `<handlers>` containing `modules="AspNetCoreModuleV2"`
   - `<aspNetCore>` element with `processPath` and `hostingModel` attributes

2. **Process execution context:**
   - In-process: `w3wp.exe` process with environment variables including `ASPNETCORE_URLS`, `ASPNETCORE_ENVIRONMENT`, or command-line args matching IIS pool name
   - Out-of-process: `dotnet.exe` or `*.exe` child process under `w3wp.exe` with parent set to IIS worker, and `ASPNETCORE_PORT` environment variable set to a high ephemeral port

3. **IIS ApplicationPool association:**
   - Run `Get-IISAppPool` (PowerShell) or inspect IIS configuration files at `C:\Windows\System32\inetsrv\config\applicationHost.config`
   - Look for `<add name="AppPoolName" ... />` with associated website physical path

4. **File system markers:**
   - Presence of `.dll` files (managed assemblies) in the application directory
   - `.runtimeconfig.json` or `.runtimeconfig.json.gz` file (framework-dependent deployment metadata)
   - `appsettings.json` or `appsettings.Production.json`

5. **Executable signatures:**
   - In-process: Checking if `w3wp.exe` has loaded `aspnetcore.dll` via `tasklist /m aspnetcore` or inspecting loaded modules with tools like Process Explorer
   - Out-of-process: `dotnet.exe` or app-specific `.exe` in the process tree under `w3wp.exe`

## Version / Variant Differences

- **In-Process vs. Out-of-Process:**
  - **In-process** (default since ASP.NET Core 3.0): App code runs in `w3wp.exe`; uses `IISHttpServer`; no Kestrel; faster (no loopback proxy)
  - **Out-of-process**: App runs in separate `dotnet.exe` or self-contained `.exe`; IIS acts as reverse proxy via loopback; uses Kestrel internally; more resilient (app crash doesn't crash w3wp.exe)
  - Controlled by `hostingModel` attribute in web.config (`inprocess` or `outofprocess`)

- **Framework-dependent vs. Self-contained:**
  - **Framework-dependent**: `processPath="dotnet"` or `processPath="C:\Program Files\dotnet\dotnet.exe"`; requires .NET runtime installed
  - **Self-contained**: `processPath=".\MyApp.exe"`; includes runtime in deployment; larger deployment size
  - Both require ASP.NET Core Module (ANCM) to be installed and registered in IIS

- **ASP.NET Core Module versions:**
  - **AspNetCoreModuleV1** (legacy, IIS 7.5+): Older module, limited features
  - **AspNetCoreModuleV2** (current): Native module for ASP.NET Core 2.0+; supports in-process and out-of-process; feature-rich

- **IIS hosting bundle versions:** Each .NET version ships a specific ANCM version; ensure the bundle version matches or exceeds the target ASP.NET Core version

## Sources

- [In-process hosting with IIS and ASP.NET Core | Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/iis/in-process-hosting?view=aspnetcore-10.0)
- [Out-of-process hosting with IIS and ASP.NET Core | Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/iis/out-of-process-hosting?view=aspnetcore-9.0)
- [Host ASP.NET Core on Windows with IIS | Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/iis/?view=aspnetcore-10.0)
- [web.config file | Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/iis/web-config?view=aspnetcore-8.0)
- [ASP.NET Core Module (ANCM) for IIS | Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/aspnet-core-module?view=aspnetcore-8.0)
