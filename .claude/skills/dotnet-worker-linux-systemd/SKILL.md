---
name: dotnet-worker-linux-systemd
description: .NET Worker Service (BackgroundService) on Linux managed by systemd; detect by process name `dotnet` running a `.dll` argument, `Type=notify` in `/etc/systemd/system/*.service`, and presence of `Microsoft.Extensions.Hosting.Systemd` NuGet package signals.
---

# .NET Worker Service on Linux — systemd

## Overview

A .NET Worker Service is a long-running background application built using the .NET generic host and `BackgroundService` abstraction, deployed on Linux and managed by systemd as a native init service. This pattern is common in production environments because it provides dependency injection, structured logging, lifetime management, and tight systemd integration without requiring containers or web frameworks. Teams choose this variant when they need a daemon/service that is not an ASP.NET Core web application but still want the benefits of the .NET hosting model.

## Deployment Process

1. Create a new worker project (or convert a console app to use generic host):
   ```bash
   dotnet new worker -n MyWorker
   ```

2. Add the systemd hosting package to the project file or via:
   ```bash
   dotnet add package Microsoft.Extensions.Hosting.Systemd
   ```

3. Update `Program.cs` to call `.UseSystemd()` on the host builder:
   ```csharp
   Host.CreateDefaultBuilder(args)
       .UseSystemd()
       .ConfigureServices((context, services) =>
       {
           services.AddHostedService<Worker>();
       })
       .Build()
       .Run();
   ```

4. Publish the application to a deployment directory:
   ```bash
   dotnet publish -c Release -o /opt/myworker
   ```

5. Create a systemd service unit file at `/etc/systemd/system/myworker.service`:
   ```ini
   [Unit]
   Description=My .NET Worker Service
   After=network-online.target
   Wants=network-online.target

   [Service]
   Type=notify
   ExecStart=/usr/bin/dotnet /opt/myworker/MyWorker.dll
   SyslogIdentifier=myworker
   User=dotnetuser
   Restart=always
   RestartSec=5
   Environment=DOTNET_ROOT=/usr/lib64/dotnet

   [Install]
   WantedBy=multi-user.target
   ```

6. Reload systemd and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable myworker.service
   sudo systemctl start myworker.service
   ```

## Process Signatures

- **Process name / executable:** `dotnet` (the .NET runtime)
- **Command-line patterns:** `dotnet /opt/*/*.dll`, `dotnet /srv/*/*.dll`, or `/usr/bin/dotnet /path/to/app.dll`
- **Parent process:** `systemd` (PID 1 or `systemd` process)
- **Typical user:** dedicated service user (e.g., `dotnetuser`, `app`, or `nobody`); often non-root
- **Working directory:** application directory (e.g., `/opt/myworker`), set by systemd or inherited from ExecStart path

## File System Paths

### Linux

- **Install root:** `/opt/myworker` or `/srv/myworker` (configurable; follows OS deployment conventions)
- **Binaries:** `/opt/myworker/MyWorker.dll` (the managed assembly), `/usr/bin/dotnet` or `/usr/lib64/dotnet/dotnet` (runtime)
- **Application/deploy dir:** `/opt/myworker/` (contains the published output including `.dll`, appsettings files, and dependencies)
- **PID file:** None (systemd tracks the process; no explicit PID file is created)
- **Service unit file:** `/etc/systemd/system/myworker.service`

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `DOTNET_ROOT` | Path to the .NET runtime installation | `/usr/lib64/dotnet` or `/usr/lib/dotnet` |
| `ASPNETCORE_ENVIRONMENT` | Selects appsettings file (e.g., appsettings.Production.json) | `Production` |
| `DOTNET_ENVIRONMENT` | Generic environment selector (if not ASP.NET Core) | `Production` |
| `DOTNET_CLI_TELEMETRY_OPTOUT` | Disables telemetry reporting | `1` |
| `DOTNET_SYSTEM_GLOBALIZATION_INVARIANT` | Forces invariant globalization for reduced memory | `1` (optional) |

## Configuration Files

- **`/opt/myworker/appsettings.json`** — Application settings in JSON format; loaded by Host.CreateDefaultBuilder
- **`/opt/myworker/appsettings.Production.json`** — Environment-specific settings overrides; loaded when `ASPNETCORE_ENVIRONMENT=Production` or `DOTNET_ENVIRONMENT=Production`
- **`/etc/systemd/system/myworker.service`** — Systemd unit file that defines how the service is started, restarted, and logged

## Log Locations

- Logs are sent to systemd journal (journald); retrieve via:
  ```bash
  sudo journalctl -u myworker.service
  sudo journalctl -u myworker.service --follow
  sudo journalctl -u myworker.service -p err
  ```
- No direct file logging to `/var/log/` unless the application explicitly configures a file-based sink (e.g., via Serilog)
- `SyslogIdentifier=myworker` in the systemd unit maps logs to the identifier "myworker" in journald

## Service / Init Integration

- **Service identifier:** `myworker.service` (the name of the unit file without path)
- **Unit file path:** `/etc/systemd/system/myworker.service`
- **Service type:** `Type=notify` (the application uses `Microsoft.Extensions.Hosting.Systemd` to send readiness notification to systemd)
- **Managed via:** `systemctl start`, `systemctl stop`, `systemctl restart`, `systemctl status myworker.service`
- **Automatic startup:** Enabled via `sudo systemctl enable myworker.service`
- **Restart behavior:** Controlled by `Restart=` and `RestartSec=` directives in the unit file (e.g., `Restart=always RestartSec=5` restarts the service if it exits, with 5-second delay)

## Detection Heuristics

1. **Highest confidence:** Process name is `dotnet` with a command-line argument ending in `.dll` (e.g., `/opt/myworker/MyWorker.dll`) and parent process is `systemd`
2. **High confidence:** Environment variable `DOTNET_ROOT` is set, and the process environment contains `INVOCATION_ID` (a systemd-specific variable injected into every service process)
3. **High confidence:** Existence of `/etc/systemd/system/*.service` file containing both `Type=notify` and an `ExecStart` line with `/usr/bin/dotnet` or similar path pointing to a `.dll`
4. **Medium confidence:** Working directory contains `appsettings.json` and/or `appsettings.Production.json` files alongside the `.dll` executable
5. **Medium confidence:** Reading `/proc/[pid]/cwd` reveals the application directory, and that directory contains a `.deps.json` file (dependency manifest produced by `dotnet publish`)
6. **Medium confidence:** `SyslogIdentifier` matches the process name as recorded in `/proc/[pid]/environ` or systemd journal for the process

## Version / Variant Differences

- **.NET Core 3.0+:** `Microsoft.Extensions.Hosting.Systemd` package is available; `.UseSystemd()` extension is present
- **.NET 5.0+:** Same support; no breaking changes to the systemd hosting API
- **.NET 6.0+:** Minimal hosting model and top-level statements simplify Program.cs; `.UseSystemd()` call remains the same
- **.NET 8.0+:** No changes to systemd integration; continues to support `Type=notify` and journal logging
- **Worker template:** Available via `dotnet new worker` in .NET Core 3.1 and later; generates a `BackgroundService` subclass by default
- **Console app variant:** A traditional console application can also be deployed as a systemd service without the Worker template, as long as it integrates `Microsoft.Extensions.Hosting.Systemd` and calls `.UseSystemd()`

## Sources

- [.NET Core and systemd — .NET Blog](https://devblogs.microsoft.com/dotnet/net-core-and-systemd/)
- [dotnet new worker — Scott Hanselman's Blog](https://www.hanselman.com/blog/dotnet-new-worker-windows-services-or-linux-systemd-services-in-net-core)
- [How to run a .NET Core console app as a service using Systemd on Linux (RHEL)](https://swimburger.net/blog/dotnet/how-to-run-a-dotnet-core-console-app-as-a-service-using-systemd-on-linux)
- [Running a .NET application as a service on Linux with Systemd — Maarten Balliauw](https://blog.maartenballiauw.be/posts/2021-05-25-running-a-net-application-as-a-service-on-linux-with-systemd/)
- [Running .NET Core Applications as a Linux Service — Code Maze](https://code-maze.com/aspnetcore-running-applications-as-linux-service/)
