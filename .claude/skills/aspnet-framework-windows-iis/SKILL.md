---
name: aspnet-framework-windows-iis
description: Classic ASP.NET Framework web app (WebAPI, MVC) hosted in IIS with w3wp.exe worker processes in integrated pipeline mode; detect via process name w3wp.exe, ASPNETCORE_ENVIRONMENT or custom environment variables in applicationHost.config, and parent process svchost.exe (W3SVC).
---

# Classic ASP.NET Framework on IIS — Integrated Pipeline

## Overview

Classic ASP.NET Framework applications (ASP.NET 4.x with .NET Framework, including WebAPI and MVC frameworks) are hosted in Internet Information Services (IIS) on Windows. The IIS worker process (w3wp.exe) runs the application under an application pool with integrated pipeline mode, which unifies the IIS HTTP pipeline with the ASP.NET managed code pipeline. This is the standard enterprise deployment model for .NET Framework web applications on Windows Server and Windows.

## Deployment Process

1. Install IIS role on Windows Server (or enable IIS on Windows client):
   ```
   Install-WindowsFeature -Name Web-Server -IncludeManagementTools
   ```

2. Ensure .NET Framework runtime (4.5 or later) is installed on the host.

3. Create or verify an IIS application pool with integrated pipeline mode:
   ```
   appcmd.exe add apppool /name:MyAppPool /managedRuntimeVersion:v4.0 /managedPipelineMode:Integrated
   ```

4. Create or update a website and application binding it to the application pool:
   ```
   appcmd.exe add app /site.name:MyWebSite /path:/myapp /physicalPath:C:\inetpub\MyApp
   appcmd.exe set app "MyWebSite/myapp" /applicationPool:MyAppPool
   ```

5. Deploy the compiled ASP.NET application binaries (.dll, .pdb, .config files) to the physical path (e.g., `C:\inetpub\MyApp`).

6. Set file permissions on the physical path to allow the application pool identity to read/execute (typically `IUSR` or a custom service account).

7. Optionally set environment variables for the application pool:
   ```
   appcmd.exe set config -section:system.applicationHost/applicationPools /+"[name='MyAppPool'].environmentVariables.[name='ASPNETCORE_ENVIRONMENT',value='Production']" /commit:apphost
   ```

8. Start or recycle the application pool:
   ```
   appcmd.exe recycle apppool /apppool.name:MyAppPool
   ```

9. The w3wp.exe worker process will spawn on the first HTTP request to the application.

## Process Signatures

- **Process name / executable:** `w3wp.exe`
- **Command-line patterns:** `C:\Windows\System32\inetsrv\w3wp.exe -apppool AppPoolName` (exact args vary; apppool name is always present)
- **Parent process:** `svchost.exe` (World Wide Web Publish Service, W3SVC)
- **Typical user:** `IUSR` (built-in), or a custom service account configured on the application pool identity
- **Working directory:** Application physical path (e.g., `C:\inetpub\MyApp\`) or the IIS root; varies by configuration but typically the site's document root

## File System Paths

### Windows

- **Install root:** IIS binaries: `C:\Windows\System32\inetsrv\`; .NET Framework runtime: `C:\Program Files\dotnet\` or `C:\Windows\Microsoft.NET\Framework64\v4.0.30319\`
- **Binaries:** `C:\Windows\System32\inetsrv\w3wp.exe` (worker process), `C:\Windows\System32\inetsrv\appcmd.exe` (management tool)
- **Application/deploy dir:** Site physical path (customer-defined); default convention: `C:\inetpub\wwwroot\` for default site, `C:\inetpub\MyAppName\` for named sites
- **Configuration:** `C:\Windows\System32\inetsrv\config\applicationHost.config` (global IIS config including app pools, sites, environment variables)
- **Web.config:** Individual application configuration at `{PhysicalPath}\web.config` (per-app ASP.NET config)

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `ASPNETCORE_ENVIRONMENT` | Runtime environment (if using ASP.NET Core interop or custom detection) | `Production`, `Development`, `Staging` |
| `ASPNET_ENV` | Legacy ASP.NET environment override (custom, not standard) | `Production` |
| Custom app-defined variables | Set per application pool in applicationHost.config for app-specific settings | Any custom value |

## Configuration Files

- **`C:\Windows\System32\inetsrv\config\applicationHost.config`** — Global IIS server configuration; defines application pools (name, .NET CLR version, pipeline mode, identity), websites, bindings, and environment variables for each app pool
- **`{PhysicalPath}\web.config`** — ASP.NET application configuration (XML); connection strings, assembly bindings, HTTP modules, security settings, compilation target framework
- **`{PhysicalPath}\packages.config`** — NuGet package manifest for Visual Studio project (informational; not loaded at runtime)
- **`{PhysicalPath}\Global.asax`** — Optional ASP.NET application lifecycle file; defines Application_Start, Application_End, Session_Start events

## Log Locations

- **IIS logs:** `C:\inetpub\logs\LogFiles\W3SVC{SiteId}\` — HTTP access logs, one per day, format defined by IIS site logging settings (typically W3C Extended Log File Format)
- **Application pool recycling logs:** `C:\inetpub\logs\FailedREQLogFiles\` (if detailed failure logging is enabled)
- **ASP.NET application logs:** Application-defined; typically `C:\inetpub\{AppName}\logs\` (custom folder created by the app) or Windows Event Viewer (if the app logs to Application event log)
- **Event Viewer:** Application log contains crashes and errors from w3wp.exe or the ASP.NET runtime (source: `ASP.NET` or app name)

## Service / Init Integration

IIS is managed via the **World Wide Web Publish Service** (W3SVC), which is a Windows system service:

- **Service name (Windows Services):** `W3SVC`
- **Display name:** "World Wide Web Publishing Service"
- **Service type:** Standard Windows service (not systemd, not launchd; Windows only)
- **Start type:** Usually "Automatic" (starts on boot)
- **Start/stop commands:**
  ```
  net start w3svc          # Start the W3SVC service
  net stop w3svc /y        # Stop the W3SVC service (forces app pool recycle)
  net stop was /y          # Stop Windows Process Activation Service (also stops W3SVC)
  ```

Application pools are child entities of W3SVC; they are not independent Windows services. Recycling an app pool does not restart W3SVC.

## Detection Heuristics

**Primary signals (in order of confidence):**

1. **Process name is `w3wp.exe`** and parent process is `svchost.exe` (W3SVC) → strong indicator of IIS-hosted application
2. **Environment variables on w3wp.exe process** — check `/proc/<pid>/environ` equivalent (Windows: WMI or PowerShell) for `ASPNETCORE_ENVIRONMENT`, `ASPNET_ENV`, or custom app pool environment variables defined in `applicationHost.config`
3. **Presence of `applicationHost.config`** in `C:\Windows\System32\inetsrv\config\` containing `<applicationPool>` entries with `managedPipelineMode="Integrated"` and matching app pool name
4. **Working directory of w3wp.exe** matches a known IIS site physical path (e.g., `C:\inetpub\...`) with `web.config` present
5. **File markers in the working directory:** presence of `Global.asax`, `web.config`, `bin\*.dll` (compiled assemblies), or `App_Data\` folder

**Combination heuristic (most unambiguous):**
- Process `w3wp.exe` with parent `svchost.exe (W3SVC)` **AND** environment variable `ASPNETCORE_ENVIRONMENT` or `ASPNET_ENV` in the process environment **AND** working directory contains `web.config`

## Version / Variant Differences

- **IIS 7.0 / 7.5 (Windows Server 2008 / 2008 R2):** Uses integrated pipeline mode; environment variables not supported (introduced in IIS 10)
- **IIS 8.0 / 8.5 (Windows Server 2012 / 2012 R2):** Integrated pipeline mode standard; no native environment variable support
- **IIS 10.0 / 10.0.1 (Windows Server 2016):** Integrated pipeline mode standard; environment variables **supported** via `<environmentVariables>` in applicationHost.config
- **IIS 10.0 (Windows Server 2019+) / IIS 10.0 (Windows 10/11):** Full environment variable support; some tooling may use IIS Express (in-process variant, different binary path)

**Pipeline mode variants:**
- **Integrated:** IIS and ASP.NET pipeline unified; modern default; all versions 7.0+
- **Classic:** Legacy mode runs ASP.NET via an ISAPI filter; rarely used in new deployments; same w3wp.exe process but different internal handling

Both modes use the same `w3wp.exe` executable; pipeline mode is a runtime configuration detail, not a file-based signal.

## Sources

- [What is w3wp.exe? - IIS Worker Process Explained](https://stackify.com/w3wp-exe-iis-worker-process/)
- [Relation between w3wp.exe and IIS Application pool - Microsoft Q&A](https://learn.microsoft.com/en-us/answers/questions/583367/relation-between-w3wp-exe-and-iis-application-pool)
- [Environment Variables <environmentVariables> | Microsoft Learn](https://learn.microsoft.com/en-us/iis/configuration/system.applicationhost/applicationpools/add/environmentvariables/)
- [Understanding the w3wp.exe Process - Professional Microsoft IIS 8](https://www.oreilly.com/library/view/professional-microsoft-iis/9781118417379/c08_level1_4.xhtml)
