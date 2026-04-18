---
name: aspnetcore-windows-service
description: ASP.NET Core application registered as a Windows Service via sc.exe create and UseWindowsService(). Detection signals: executable named *.exe in Windows registry HKLM\SYSTEM\CurrentControlSet\Services, parent process services.exe, event logs in Windows Application Event Log with source matching app namespace.
---

# ASP.NET Core on Windows — Windows Service

## Overview
ASP.NET Core applications can be deployed as Windows Services without IIS, using the `UseWindowsService()` method and the Windows Service Control Manager (sc.exe). This pattern is common in enterprise environments for background services, worker services, and web APIs that require no external HTTP server. The service runs as a child of services.exe and logs to the Windows Application Event Log rather than file-based logs by default.

## Deployment Process

1. **Publish the application** as a self-contained executable (.exe):
   ```powershell
   dotnet publish --configuration Release --output "C:\publish\MyService" --runtime win-x64 --self-contained
   ```

2. **Install the NuGet package** (if not already included):
   ```
   Microsoft.Extensions.Hosting.WindowsServices
   ```

3. **Modify Program.cs** to call `AddWindowsService()`:
   ```csharp
   HostApplicationBuilder builder = Host.CreateApplicationBuilder(args);
   builder.Services.AddWindowsService(options =>
   {
       options.ServiceName = "MyService";
   });
   // ... register hosted services ...
   IHost host = builder.Build();
   host.Run();
   ```

4. **Register the service** with the Windows Service Control Manager:
   ```powershell
   sc.exe create "MyService" binpath= "C:\publish\MyService\MyService.exe" DisplayName= "My Service" obj= LocalService start= auto
   ```
   (Run PowerShell as Administrator.)

5. **Start the service**:
   ```powershell
   sc.exe start "MyService"
   ```

## Process Signatures
- **Process name / executable:** The application's `.exe` filename (e.g., `MyService.exe`, `TodosService.exe`). The process binary is located in the directory specified by `binpath` during registration.
- **Command-line patterns:** The executable invocation may include optional arguments such as `--contentRoot C:\Path\To\App`. No embedded runtime; the executable is self-contained or delegates to `dotnet.exe C:\Path\To\App.dll`.
- **Parent process:** `services.exe` (the Windows Service Control Manager host process).
- **Typical user:** `SYSTEM` (default), `LocalService`, `NetworkService`, or a custom user account specified during `sc.exe create`.
- **Working directory:** By default, `C:\Windows\System32` or `C:\Windows\SysWOW64`. Can be overridden via `--contentRoot` argument or app configuration.

## File System Paths

### Windows
- **Install root:** User-defined at `sc.exe create` time; typical locations include `C:\Program Files\MyApp`, `C:\Services\MyService`, or `C:\publish\MyService`.
- **Binaries:** `<install-root>\MyService.exe` (or the project's `.exe` name).
- **Application/deploy dir:** Same as install root; all assemblies, configuration files, and dependencies are in this directory.
- **Service registry entry:** `HKLM\SYSTEM\CurrentControlSet\Services\MyService` (where `MyService` is the service name passed to `sc.exe create`). The `ImagePath` value contains the full path to the `.exe`.

## Environment Variables
| Variable | Purpose | Typical value |
|---|---|---|
| `ASPNETCORE_ENVIRONMENT` | Selects appsettings file variant (Development, Staging, Production) | `Production` |
| `DOTNET_*` | Runtime configuration (e.g., `DOTNET_GCSERVER`, `DOTNET_MULTILEVEL_LOOKUP`) | `1` |
| `ASPNETCORE_URLS` | Kestrel binding addresses (if hosting HTTP directly) | `http://localhost:5000` |

These are typically set in `appsettings.json` or environment-specific `appsettings.{Environment}.json`, not as system environment variables.

## Configuration Files
- **appsettings.json** — Application configuration, logging levels, and event log settings. Located in the install root directory. Example:
  ```json
  {
    "Logging": {
      "LogLevel": {
        "Default": "Information"
      },
      "EventLog": {
        "SourceName": "MyService",
        "LogName": "Application",
        "LogLevel": {
          "Default": "Warning"
        }
      }
    }
  }
  ```
- **appsettings.{Environment}.json** — Environment-specific overrides (e.g., `appsettings.Production.json`).

## Log Locations
- **Windows Application Event Log** — All logging by default goes to the Application Event Log accessible via **Event Viewer** > **Windows Logs** > **Application**. Log source name is configurable in `appsettings.json` under `EventLog.SourceName` (default matches the app namespace or `.ServiceName` set in `AddWindowsService()`).
- **No file-based logs by default** — Unless explicitly configured via Serilog, NLog, or another file-based logging provider. File logging must be added to the logging pipeline in Program.cs.

## Service / Init Integration
Windows Service Control Manager (sc.exe). The service is registered in the registry at `HKLM\SYSTEM\CurrentControlSet\Services\<ServiceName>`, where `<ServiceName>` is the name passed to `sc.exe create`. Management commands:
- **Create:** `sc.exe create "ServiceName" binpath= "C:\Path\To\App.exe"`
- **Start:** `sc.exe start "ServiceName"`
- **Stop:** `sc.exe stop "ServiceName"`
- **Delete:** `sc.exe delete "ServiceName"`
- **Query configuration:** `sc.exe qc "ServiceName"`
- **Query failure/recovery:** `sc.exe qfailure "ServiceName"`
- **Set failure recovery:** `sc.exe failure "ServiceName" reset= <seconds> actions= restart/<delay>/restart/<delay>/run/<delay>`

The service can also be managed via the Services MMC snap-in (`services.msc`).

## Detection Heuristics
- **Registry check:** Presence of service key in `HKLM\SYSTEM\CurrentControlSet\Services\<ServiceName>` with `ImagePath` pointing to a `.exe` file in a user-defined directory.
- **Process parent:** Parent process is `services.exe` (PID 1 or a system service host).
- **Event Log source:** Look for a source in the Application Event Log matching the `ServiceName` or app namespace.
- **Executable path pattern:** `.exe` file whose directory is not `C:\Windows\System32` and is registered in the Services registry hive.
- **Loaded assembly check (if accessible):** Presence of `Microsoft.Extensions.Hosting.WindowsServices.dll` in the application directory.
- **Command-line arguments:** Service executable may include `--contentRoot` or other host-configuration arguments.

## Version / Variant Differences
- **.NET 6+:** Default exception behavior changed from `Ignore` to `StopHost`; services must call `Environment.Exit(1)` in catch blocks to enable SCM recovery options.
- **.NET 5 and earlier:** Default exception behavior is `Ignore`, which can result in zombie processes if unhandled exceptions occur.
- **Single-file vs. framework-dependent:** Single-file executables (set `<PublishSingleFile>true</PublishSingleFile>` in .csproj) have no external dependencies; framework-dependent deployments require the .NET runtime installed separately.
- **ASP.NET Core vs. Worker Service:** ASP.NET Core services use `Host.CreateApplicationBuilder()` and typically include web hosting; Worker Services use `Host.CreateDefaultBuilder()` and `BackgroundService`. Both use `AddWindowsService()`.

## Sources
- [Host ASP.NET Core in a Windows Service - Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/windows-service?view=aspnetcore-10.0)
- [Create Windows Service using BackgroundService - .NET - Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/extensions/windows-service)
- [Running .NET Core Applications as a Windows Service - Code Maze](https://code-maze.com/aspnetcore-running-applications-as-windows-service/)
