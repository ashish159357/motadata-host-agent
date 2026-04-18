---
name: dotnet-framework-windows-service
description: .NET Framework Windows Service running on Windows using ServiceBase or TopShelf, identified by .exe process running under SYSTEM or NetworkService account, registered in HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services registry hive, with parent process Services.exe.
---

# .NET Framework Windows Service on Windows — Classic .NET Framework Windows Service (ServiceBase / TopShelf)

## Overview

A .NET Framework Windows Service is a long-running background service deployed on Windows that runs under the Service Control Manager (SCM). Teams choose this pattern for hosting backend services, scheduled tasks, and integrations that require always-on availability. ServiceBase (built into .NET Framework) and TopShelf (a popular open-source wrapper) are the two primary frameworks. This pattern is still common in enterprise Windows environments, particularly for legacy and line-of-business applications.

## Deployment Process

1. **Build the service:** Compile the .NET Framework console application with a `ServiceBase`-derived class or TopShelf `HostFactory.Run()` configuration.
2. **Create installer:** Add `ServiceInstaller` and `ServiceProcessInstaller` components to the project (ServiceBase) or use TopShelf's `RunAs*` fluent API.
3. **Install the service:** Run `installutil.exe MyService.exe` (ServiceBase) or `MyService.exe install` (TopShelf) from an elevated command prompt.
4. **Configure startup:** Set `StartType` to `Automatic`, `Manual`, or `Disabled` via the installer or Services.msc snap-in.
5. **Start the service:** Use `net start MyServiceName` or start from Services.msc.
6. **Uninstall (if needed):** Run `installutil.exe /u MyService.exe` or `MyService.exe uninstall`.

## Process Signatures

- **Process name / executable:** `<ServiceName>.exe` (e.g., `MyNewService.exe`, `WindowsServiceWithTopshelf.exe`)
- **Command-line patterns:** Executable name only, e.g. `C:\Program Files\MyService\MyNewService.exe`, or `C:\opt\service\product.exe` (TopShelf). No arguments visible to Process Monitor for running service instances.
- **Parent process:** `services.exe` (Windows Service Control Manager)
- **Typical user:** `SYSTEM`, `NT AUTHORITY\NETWORK SERVICE`, or custom domain account (configured via `ServiceProcessInstaller.Account`)
- **Working directory:** Installation directory (e.g., `C:\Program Files\MyService` or `C:\opt\service`); controlled by `ImagePath` registry value

## File System Paths

### Windows

- **Install root:** `C:\Program Files\<ServiceName>` or `C:\Program Files (x86)\<ServiceName>` (ServiceBase); custom path for TopShelf (e.g., `C:\opt\<service>`).
- **Binaries:** `C:\Program Files\<ServiceName>\<ServiceName>.exe`, `.dll` assemblies in same directory.
- **Application/deploy dir:** Same as install root; configuration files (`.config`, `.json`) and supporting DLLs colocated.

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `ASPNETCORE_ENVIRONMENT` | (Rare in Framework; present in Core-based services) | `Production`, `Development` |
| `JAVA_HOME`, `PYTHON_PATH` | (Not applicable to .NET Framework services; these indicate other runtimes) | N/A |

## Configuration Files

- **`<ServiceName>.exe.config`** — XML configuration file for app settings, logging, and database connection strings; located in install root (e.g., `C:\Program Files\MyService\MyNewService.exe.config`). Standard .NET Framework ConfigurationManager reads from this file.
- **`appsettings.json`** — (TopShelf or modern .NET Framework services using Microsoft.Extensions.Configuration) optional JSON config file in install root.
- **Registry: `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>`** — Windows registry hive storing service metadata. Key sub-values:
  - `ImagePath`: Full path to executable (e.g., `C:\Program Files\MyService\MyNewService.exe`).
  - `Start`: Startup type (0=Boot, 1=System, 2=Automatic, 3=Manual, 4=Disabled).
  - `DisplayName`: Human-readable service name.
  - `Description`: Service description string.
  - `ObjectName`: Account SID or account name (e.g., `NT AUTHORITY\SYSTEM`).

## Log Locations

- **Windows Event Log:** Services typically log to `Applications and Services Logs\<CustomLogName>` or `Windows Logs\Application`. Source name is registered via `EventLog.CreateEventSource()` in ServiceBase constructor. Example: `MyNewLog` event source → `Windows Logs\Application` or custom log.
- **File-based logs:** Application-specific text or JSON logs in install root or subdirectory (e.g., `C:\Program Files\MyService\logs\app.log`). No standard rotation pattern; application-dependent.
- **Example Event Viewer path:** Event Viewer → `Windows Logs\Application` → filter by Source `MySource` (as configured in service code).

## Service / Init Integration

.NET Framework Windows Services are registered with and supervised by **Windows Service Control Manager (SCM)**, not systemd. The service entry is stored in the Windows registry under `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>`.

- **Service name:** User-defined via `ServiceInstaller.ServiceName` (ServiceBase) or `service.ServiceName(x => "MyServiceName")` (TopShelf); used in `net start <name>` and `net stop <name>` commands.
- **Start the service:** `net start <ServiceName>` or right-click → Start in Services.msc.
- **Stop the service:** `net stop <ServiceName>` or right-click → Stop in Services.msc.
- **View service status:** `services.msc` or `sc query <ServiceName>` (command-line utility).
- **Retrieve startup type / account:** `sc qc <ServiceName>` (shows full service config including IMAGE_PATH, SERVICE_TYPE, START_TYPE, DISPLAY_NAME).

## Detection Heuristics

1. **Registry lookup:** Check `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services` for a subkey matching the service name; if present, read `ImagePath` value to confirm it is a `.exe` file.
2. **Parent process:** Running process with parent `services.exe` (PID 1) is a strong signal for a Windows service.
3. **Executable location:** `.exe` files in `C:\Program Files\`, `C:\Program Files (x86)\`, or custom install directories that match a registry service entry.
4. **File-system markers:** Presence of `.exe.config`, `.pdb` (debug symbols), or `.deps.json` in the same directory as `.exe`.
5. **Event Log sources:** Use `PowerShell Get-EventLog -LogName Application | Where-Object {$_.Source -eq "MySource"}` or Event Viewer to identify registered event sources.
6. **Account context:** Services running under `NT AUTHORITY\SYSTEM`, `NT AUTHORITY\NETWORK SERVICE`, or other high-privilege accounts are likely Windows services.

## Version / Variant Differences

- **ServiceBase (native .NET Framework):** Uses `System.ServiceProcess.ServiceBase` class; installation via `installutil.exe` (a .NET Framework tool). Requires explicit `ServiceInstaller` and `ServiceProcessInstaller` classes. More verbose; older pattern (2000s–2010s).
- **TopShelf:** Simplifies service creation with fluent API (`HostFactory.Run()`); installation via `<executable>.exe install` command (self-hosted). No separate installer tool required. Modern, lightweight wrapper (2010s–present). Both ServiceBase and TopShelf compile to standard `.exe` files; detection heuristics are identical.
- **.NET Framework vs. .NET Core:** This skill covers .NET Framework (4.0–4.8.x) running on Windows. .NET Core / .NET 5+ services use different detection signals (e.g., `dotnet.exe` host, `appsettings.json`). Distinguish by checking if the executable is the service binary itself (Framework) or a thin wrapper calling `dotnet.exe` (Core).

## Sources

- [Developing Windows Service Applications - .NET Framework | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/framework/windows-services/)
- [Tutorial: Create a Windows service app - .NET Framework | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/framework/windows-services/walkthrough-creating-a-windows-service-application-in-the-component-designer)
- [Creating Windows Service In .NET with Topshelf | C# Corner](https://www.c-sharpcorner.com/article/creating-windows-service-in-net-with-topshelf/)
- [GitHub - Topshelf/Topshelf: An easy service hosting framework for building Windows services using .NET](https://github.com/Topshelf/Topshelf)
