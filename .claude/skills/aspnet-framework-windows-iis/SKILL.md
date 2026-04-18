---
name: aspnet-framework-windows-iis
description: ASP.NET Framework application hosted in IIS on Windows; process w3wp.exe running under IIS AppPool identity; application root in C:\inetpub\wwwroot\; web.config and applicationHost.config present.
---

# ASP.NET Framework on Windows — IIS

## Overview

ASP.NET Framework applications (classic .NET Framework 4.x running ASP.NET WebForms, MVC, or WebAPI) are hosted directly in Internet Information Services (IIS) on Windows Server or Windows client OS. IIS is Microsoft's native web server and the standard deployment pattern for on-premises .NET Framework shops. The application runs in an IIS worker process (w3wp.exe) under an application pool identity. This variant is extremely common in enterprise production environments and remains heavily used despite the push toward ASP.NET Core.

## Deployment Process

1. **Install IIS and .NET Framework prerequisites:**
   - Open Control Panel → Programs → Programs and Features → Turn Windows features on or off
   - Expand Internet Information Services → World Wide Web Services → Application Development Features
   - Check ASP.NET 4.7 (or the corresponding version for your .NET Framework target)
   - Confirm World Wide Web Services and IIS Management Console are selected

2. **Verify IIS configuration:**
   - Open IIS Manager (press Windows+R, type `inetmgr`)
   - In Application Pools, confirm DefaultAppPool (or your custom pool) is set to .NET CLR v4.0.30319
   - If not, right-click the pool, select Basic Settings, and change .NET CLR version accordingly

3. **Create or configure the application pool:**
   - In IIS Manager, right-click Application Pools and select Add Application Pool
   - Provide a name (e.g., `ContosoUniversityPool`)
   - Select .NET CLR version 4.0.30319 (or higher if targeting later Framework versions)
   - Set process model identity (ApplicationPoolIdentity, NetworkService, or custom account)
   - Configure process model recycling, timeout, and shutdown settings as needed

4. **Create the IIS website/application:**
   - In IIS Manager, right-click Sites and select Add Website
   - Provide Site name (e.g., `ContosoUniversity`)
   - Set Physical path to the application root (e.g., `C:\inetpub\wwwroot\ContosoUniversity`)
   - Configure hostname, port, and SSL bindings
   - Assign the application to the correct application pool

5. **Deploy application binaries:**
   - Copy the compiled application (from `bin\Release\<Framework>\publish` in Visual Studio) to the physical path
   - Ensure web.config is present in the application root
   - Run any SQL Server setup scripts if the application requires database access

6. **Grant database permissions (if applicable):**
   - If the application accesses SQL Server Express or SQL Server, run a T-SQL script to grant the application pool identity (e.g., `IIS APPPOOL\ContosoUniversityPool`) the necessary database permissions:
     ```sql
     IF NOT EXISTS (SELECT name FROM sys.server_principals WHERE name = 'IIS APPPOOL\ContosoUniversityPool')
     BEGIN
         CREATE LOGIN [IIS APPPOOL\ContosoUniversityPool] 
           FROM WINDOWS WITH DEFAULT_DATABASE=[master]
     END
     GO
     CREATE USER [ContosoUniversityUser] 
       FOR LOGIN [IIS APPPOOL\ContosoUniversityPool]
     GO
     EXEC sp_addrolemember 'db_owner', 'ContosoUniversityUser'
     GO
     ```

7. **Verify and test:**
   - Navigate to the site in a web browser (e.g., `http://localhost/ContosoUniversity`)
   - Confirm the application loads and database operations work

## Process Signatures

- **Process name / executable:** `w3wp.exe` (located in `C:\Windows\System32\inetsrv\`)
- **Command-line patterns:** 
  - `C:\Windows\System32\inetsrv\w3wp.exe -s <SiteName> -h <ApplicationHostConfigPath>` (typical launch by WAS)
  - May include additional flags such as `-w <RootWebConfigPath>` or `-debug`
- **Parent process:** Windows Process Activation Service (WAS) or W3SVC (World Wide Web Publishing Service)
- **Typical user:** `IIS APPPOOL\DefaultAppPool`, `IIS APPPOOL\<PoolName>`, `NETWORK SERVICE`, or a custom domain/local account
- **Working directory:** The physical path of the application (e.g., `C:\inetpub\wwwroot\ContosoUniversity\`)

## File System Paths

### Windows

- **Install root:** `C:\inetpub\wwwroot\` (default IIS root for websites)
- **Binaries:** 
  - Compiled .NET Framework assemblies in `C:\inetpub\wwwroot\<AppName>\bin\` (e.g., `MyApp.dll`)
  - IIS worker process binary: `C:\Windows\System32\inetsrv\w3wp.exe`
- **Application/deploy dir:** `C:\inetpub\wwwroot\<AppName>\` (contains .aspx, .ascx, web.config, bin/, and App_Data/ directories)
- **Logs:** 
  - IIS logs (access/request logs): `C:\inetpub\logs\LogFiles\W3SVC<SiteID>\` (named `exYYMMDD.log`)
  - Application-generated logs (if configured): typically in `C:\inetpub\wwwroot\<AppName>\App_Data\` or a custom location specified in web.config

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `ASPNETCORE_ENVIRONMENT` | (ASP.NET Core only; not used in Framework) | N/A for Framework |
| `PYTHONPATH`, `NODEJS_PATH` | (not applicable) | N/A |
| `DOTNET_ROOT` | (optional, rarely set in Framework) | C:\Program Files\dotnet\ |
| Variables set by WAS/W3SVC services | Inherited by w3wp.exe processes | User-defined at service registry level (HKLM:SYSTEM\CurrentControlSet\Services\W3SVC or WAS) |

Custom environment variables can be configured per application pool or application in `applicationHost.config` within `<environmentVariables>` sections.

## Configuration Files

- **web.config** — Primary application configuration file (XML format); located in the application root (e.g., `C:\inetpub\wwwroot\ContosoUniversity\web.config`). Contains `<system.web>`, `<system.webServer>`, `<appSettings>`, `<connectionStrings>`, and custom application sections. IIS and ASP.NET Framework read this on every request.
- **applicationHost.config** — IIS-wide configuration file defining sites, application pools, modules, and handlers; located at `C:\Windows\System32\inetsrv\config\applicationHost.config`. Contains the `<system.applicationHost>` section with all site and pool definitions. WAS reads this when creating/recycling worker processes.
- **Machine.config** — System-wide .NET Framework configuration (rarely edited directly); located at `C:\Windows\Microsoft.NET\Framework\v4.0.30319\Config\Machine.config` (32-bit) or `C:\Windows\Microsoft.NET\Framework64\v4.0.30319\Config\Machine.config` (64-bit). ASP.NET Framework reads this as a base layer before web.config.

## Log Locations

- **IIS access/request logs:** `C:\inetpub\logs\LogFiles\W3SVC<SiteID>\exYYMMDD.log` (daily rollover by default; format configurable as W3C Extended Log Format, IIS Log Format, or NCSA Common Log Format)
- **IIS Failed Request Tracing (FREB):** `C:\inetpub\logs\FailedReqLogFiles\<SiteName>\` (if FREB is enabled for debugging specific requests)
- **Application-generated logs:** Typically in `C:\inetpub\wwwroot\<AppName>\App_Data\` or paths configured in web.config (e.g., ELMAH error logs, custom log directories)
- **Event logs:** Windows Event Viewer under Windows Logs → System (WAS/W3SVC events) and Windows Logs → Application (ASP.NET runtime exceptions)

## Service / Init Integration

ASP.NET Framework applications in IIS are not managed by systemd (Linux) or Windows Task Scheduler. Instead, they are supervised by **Windows Process Activation Service (WAS)** and **World Wide Web Publishing Service (W3SVC)**:

- **W3SVC (World Wide Web Publishing Service):** Windows service (`net start w3svc`, `net stop w3svc`) that manages HTTP.sys and IIS configuration; typically set to Automatic startup.
- **WAS (Windows Process Activation Service):** Windows service (`net start was`, `net stop was`) that creates and recycles worker processes (w3wp.exe) based on `applicationHost.config` and application pool settings; typically set to Automatic startup.

To manage the application pool and restart the application:
- Use IIS Manager: right-click the application pool → Recycle
- Use `appcmd.exe` command-line tool:
  ```cmd
  C:\Windows\System32\inetsrv\appcmd.exe recycle apppool /apppool.name:"ContosoUniversityPool"
  ```
- Restart W3SVC service:
  ```cmd
  net stop /y was
  net start w3svc
  ```

## Detection Heuristics

1. **Process name:** Look for `w3wp.exe` process running under an `IIS APPPOOL\<PoolName>` or `NETWORK SERVICE` user.
2. **Executable path:** Confirm the executable is located at `C:\Windows\System32\inetsrv\w3wp.exe`.
3. **Parent process:** Verify the parent process is WAS or W3SVC.
4. **Working directory:** Check that the working directory is `C:\inetpub\wwwroot\<AppName>\` (or another configured IIS physical path).
5. **Configuration files:** Confirm presence of `web.config` in the application root and an entry in `applicationHost.config` for the corresponding site/application pool.
6. **File extensions:** Look for `.aspx`, `.ascx`, `.asmx`, `.asp` files (ASP.NET Framework indicators) in the application directory.
7. **Assembly paths:** In the application's `bin\` directory, look for .NET Framework assemblies (e.g., `System.Web.dll`, `System.Web.Mvc.dll`, `System.Web.WebPages.dll`) confirming Framework (not Core) usage.
8. **Registry:** Check `HKLM:SYSTEM\CurrentControlSet\Services\W3SVC` and `HKLM:SYSTEM\CurrentControlSet\Services\WAS` for service status and configuration.

## Version / Variant Differences

- **ASP.NET Framework 4.5 through 4.8.x:** All run on the same CLR version (.NET CLR v4.0.30319) configured in IIS application pools. Minor Framework version differences (4.5, 4.6, 4.7, 4.8) do not change process signatures or deployment topology.
- **ASP.NET WebForms vs. MVC vs. WebAPI:** All compile to the same .NET Framework assemblies and run identically in IIS worker processes. Detection cannot distinguish between these architectures by process signature alone; file inspection (presence of `Global.asax`, `packages.config`, NuGet references) is required.
- **IIS 7.0+ (Windows Vista/Server 2008+) vs. IIS 6.0 (Windows Server 2003):** IIS 6.0 and earlier use different process and isolation models (isapi_wp.exe, application isolation levels). This skill focuses on IIS 7.0+ integrated pipeline; IIS 6.0 is end-of-life.
- **32-bit vs. 64-bit processes:** IIS application pools can be configured to run in 32-bit mode (`Enable 32-Bit Applications` setting) or 64-bit mode. Both run `w3wp.exe` with identical process signatures; the bitness is determined by the platform target of the application and the pool configuration.

## Sources

- [Deploy Applications Built on .NET Framework - Microsoft Learn](https://learn.microsoft.com/en-us/troubleshoot/developer/dotnet/framework/installation/deploy-applications)
- [ASP.NET Web Deployment using Visual Studio: Deploying to IIS - Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/web-forms/overview/deployment/visual-studio-web-deployment/deploying-to-iis)
- [w3wp.exe | IIS Worker Process | STRONTIC](https://strontic.github.io/xcyclopedia/library/w3wp.exe-A93EC9C3999C4F798B270A4B5C5300A1.html)
- [What is w3wp.exe? - IIS Worker Process Explained - Stackify](https://stackify.com/w3wp-exe-iis-worker-process/)
- [Setting application environment variables in IIS without restarts - Andrew Lock](https://andrewlock.net/setting-environment-variables-in-iis-and-avoiding-app-pool-restarts/)
