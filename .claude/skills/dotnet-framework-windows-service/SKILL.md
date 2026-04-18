---
name: dotnet-framework-windows-service
description: Classic .NET Framework (4.x) Windows Service running under Windows Service Control Manager; detect by executable inheriting ServiceBase, service installed via InstallUtil.exe or sc.exe, process parent is services.exe (PID 1), and registry entry in HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services.
---

# Classic .NET Framework Windows Service on Windows — SystemServiceBase/TopShelf

## Overview

Classic .NET Framework Windows Services are long-running .NET Framework 4.x applications that run under Windows Service Control Manager (SCM) as system services. They inherit from `System.ServiceProcess.ServiceBase` or use frameworks like TopShelf to abstract the service hosting details. This is the traditional (pre-.NET Core) approach to building background services on Windows; teams choose this pattern for legacy applications that must run as services with automatic startup, persistence across reboots, and tight Windows integration. It remains common in enterprise environments with large existing .NET Framework codebases.

## Deployment Process

1. **Develop and build the service application:** Create a Console Application (.NET Framework 4.x) with a class inheriting from `System.ServiceProcess.ServiceBase`, or use TopShelf framework. Add a `ProjectInstaller` class with `ServiceInstaller` and `ServiceProcessInstaller` attributes set with `RunInstaller(true)`.
   ```csharp
   [RunInstaller(true)]
   public class ProjectInstaller : Installer { ... }
   ```

2. **Compile the executable:** Build the .exe (e.g., `MyService.exe`) in Release mode.

3. **Install using InstallUtil.exe:**
   ```cmd
   C:\Windows\Microsoft.NET\Framework\v4.0.30319\installutil.exe MyService.exe
   ```
   Or uninstall with:
   ```cmd
   C:\Windows\Microsoft.NET\Framework\v4.0.30319\installutil.exe /u MyService.exe
   ```

4. **Alternatively, install via sc.exe (net.exe):**
   ```cmd
   sc.exe create MyService binPath= "C:\Path\To\MyService.exe"
   sc.exe config MyService start= auto
   net start MyService
   ```

5. **Verify installation in Services:** Open `services.msc` (Services Console) and confirm the service is listed and set to Automatic or Manual startup.

6. **Start the service:**
   ```cmd
   net start MyService
   ```
   Or via sc:
   ```cmd
   sc.exe start MyService
   ```

## Process Signatures

- **Process name / executable:** Application-specific .exe name (e.g., `MyService.exe`, `ServiceHost.exe`). The executable runs directly from the install directory, not via an interpreter.
- **Command-line patterns:** Service executables run with no command-line arguments or minimal arguments. Launch line appears in Process explorer as the full path to the .exe, e.g., `C:\Program Files\MyApp\MyService.exe`.
- **Parent process:** `services.exe` (Windows Service Control Manager, typically PID 1 or parent of all services).
- **Typical user:** `SYSTEM`, `LOCAL SERVICE`, `NETWORK SERVICE`, or a custom domain/local user account configured in `ServiceProcessInstaller.Account`.
- **Working directory:** The directory containing the .exe file is the implicit working directory; common locations include `C:\Program Files\AppName\`, `C:\Program Files (x86)\AppName\`, or custom install paths.

## File System Paths

### Windows

- **Install root:** `C:\Program Files\<AppName>\` or `C:\Program Files (x86)\<AppName>\` (standard for msi/installer deployments); custom paths possible (e.g., `C:\Services\<AppName>\`).
- **Binaries:** `C:\Program Files\<AppName>\<ServiceName>.exe` (main service executable); may also contain `*.dll` assemblies and dependencies.
- **Application/deploy dir:** Same as install root; contains the .exe, configuration files (e.g., `app.config`), and library files.
- **Configuration files:** `<InstallRoot>\<ServiceName>.exe.config` (XML application configuration file); may reference external config in `C:\ProgramData\<AppName>\` or registry.
- **Logs:** Event Log entries written to Windows Event Viewer under **Applications and Services Logs** → custom log name (e.g., **MyNewLog**); optional text/file-based logs in `C:\ProgramData\<AppName>\Logs\` or `%TEMP%\`.

### Registry

- **Service registration:** `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>` (created by InstallUtil or sc.exe).
- **ImagePath:** Registry value at `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>\ImagePath` contains the full path to the .exe.
- **Start type:** Registry value `Start` (0=Boot, 1=System, 2=Auto, 3=Manual, 4=Disabled).
- **Parameters subkey:** `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>\Parameters` may store service-specific configuration.

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `DOTNET_ROOT` | .NET Framework installation directory | `C:\Windows\Microsoft.NET\Framework\v4.0.30319` (32-bit) or `Framework64` (64-bit) |
| Application-specific vars | Service may read custom env vars set at install time or in registry | Configured via `sc.exe environment` or installer class |

Note: Environment variables are not primary indicators for .NET Framework service detection; instead, examine the executable type (managed .NET assembly) and service registration.

## Configuration Files

- **app.config** — Standard .NET application configuration file (XML). Located at `<InstallRoot>\<ServiceName>.exe.config`. Contains AppSettings, ConnectionStrings, and custom config sections for service behavior.
- **installer class manifest** — Embedded in the .exe as metadata (RunInstallerAttribute); not a separate file but visible via .NET reflection tools.
- **Registry Parameters subkey** — Optional; service may read configuration from `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>\Parameters`.

## Log Locations

- **Windows Event Log** — Primary location for service diagnostics. Entries appear in **Event Viewer** under **Applications and Services Logs** → service's custom log name (e.g., **MyNewLog**). Entries include OnStart, OnStop, and custom events written via `System.Diagnostics.EventLog` class.
- **Optional file-based logs** — Service may write to text files in `C:\ProgramData\<AppName>\Logs\`, `%LOCALAPPDATA%\<AppName>\`, or the application's install directory. No standard location; depends on service implementation.

## Service / Init Integration

Windows Service Control Manager (SCM) manages the service lifecycle. Registration occurs in the Windows Registry at `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>`.

- **Service name:** The name registered with SCM (e.g., `MyService`). This is distinct from the display name and is used by `sc.exe`, `net.exe`, and Service Manager.
- **Display name:** Friendly name shown in Services console; set via `ServiceInstaller.DisplayName`.
- **Start type:** Configured in registry `Start` value:
  - `0` = Boot start
  - `1` = System start
  - `2` = Auto start (startup=Automatic)
  - `3` = Manual
  - `4` = Disabled
- **Service start/stop:** Initiated by SCM or user via `net start <ServiceName>`, `net stop <ServiceName>`, `sc.exe start <ServiceName>`, or Services console.

## Detection Heuristics

The following signals unambiguously identify a .NET Framework Windows Service:

1. **Process parent is `services.exe`** — The service runs under Windows Service Control Manager, not as a standalone console application.
2. **Executable is a managed .NET assembly** — The .exe file has a .NET Framework (4.x) header; the assembly contains `System.ServiceProcess.ServiceBase` type or calls to `ServiceBase.Run()` in the Main method. Use `System.Reflection` or tools like `ildasm.exe` to verify.
3. **Registry service entry exists** — Entry in `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\<ServiceName>` with `ImagePath` pointing to the executable.
4. **Service descriptor in installer class** — The assembly contains a class with `[RunInstaller(true)]` attribute and an `Installer` type with `ServiceInstaller` and `ServiceProcessInstaller` components.
5. **No command-line interpreter in parent chain** — Process tree shows service.exe → MyService.exe; no intermediate cmd.exe, powershell.exe, or script host.
6. **Event Log source registered** — Custom event log source or default Application log shows entries with source matching the service name or a configured event source (visible in Event Viewer or registry under `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\EventLog`).

**Detection combination:** If a process is a child of `services.exe`, the executable is a .NET Framework managed assembly, and a registry entry for that service name exists with matching ImagePath, it is a Classic .NET Framework Windows Service.

## Version / Variant Differences

- **.NET Framework 4.0, 4.5, 4.6, 4.7, 4.8:** All use the same `System.ServiceProcess` namespace and ServiceBase API. ServiceBase behavior and capabilities are consistent across these versions.
- **TopShelf framework variant:** Services built with TopShelf abstract ServiceBase; they generate a ServiceInstaller at runtime or install via TopShelf's own `install` command (e.g., `MyService.exe install -servicename=MyService`). The resulting registry entry and process behavior remain identical to manual ServiceBase implementation.
- **Account types:**
  - `LocalSystem` — Highest privilege; can write to event log and access system resources.
  - `LocalService` — Reduced privilege; limited file access; cannot write to event log by default.
  - `NetworkService` — Network access; reduced local privilege.
  - Custom user account — Service runs under domain or local user credentials; registry entry includes `ObjectName` (service account principal).
- **Installation method:** InstallUtil.exe vs. sc.exe produce equivalent registry entries. sc.exe is preferred on modern Windows because InstallUtil.exe may not be available in slim installations.

## Sources

- [How to: Install and uninstall Windows services - .NET Framework | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/framework/windows-services/how-to-install-and-uninstall-services)
- [Installutil.exe (Installer Tool) - .NET Framework | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/framework/tools/installutil-exe-installer-tool)
- [Tutorial: Create a Windows service app - .NET Framework | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/framework/windows-services/walkthrough-creating-a-windows-service-application-in-the-component-designer)
- [Install and configure a Windows Service from the command line](https://makolyte.com/install-and-configure-a-windows-service-from-the-command-line/)
- [Creating Windows Service In .NET with Topshelf | C# Corner](https://www.c-sharpcorner.com/article/creating-windows-service-in-net-with-topshelf)
