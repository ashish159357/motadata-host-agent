---
name: dotnet-worker-linux-systemd
description: .NET Worker Service (generic host with BackgroundService) on Linux supervised by systemd with UseSystemd() integration; detect by `dotnet` or app-name executable as child of systemd with DOTNET_ROOT env var, Type=notify in /etc/systemd/system/*.service, and Microsoft.Extensions.Hosting.Systemd assembly loaded.
---

# .NET Worker Service on Linux — systemd Supervised

## Overview

A .NET Worker Service is a long-running, non-HTTP background application built using the generic host with `BackgroundService` infrastructure, deployed on Linux and supervised by systemd. The application integrates with systemd using the `Microsoft.Extensions.Hosting.Systemd` NuGet package and `UseSystemd()` in its host builder, enabling systemd to manage process lifecycle, receive startup/shutdown notifications (`Type=notify`), and route application logs to journald. This is the standard pattern for .NET background services, scheduled tasks, and daemon-like workloads on Linux in production environments.

## Deployment Process

1. Create a new .NET Worker Service project using `dotnet new worker -n MyWorkerApp`.
2. Add the `Microsoft.Extensions.Hosting.Systemd` NuGet package to the project.
3. Call `.UseSystemd()` in `Program.cs` on the `IHostBuilder` (no-op on non-systemd platforms).
4. Implement the `Worker` class extending `BackgroundService` with business logic in `ExecuteAsync()`.
5. Publish the application as a self-contained executable: `dotnet publish -c Release -r linux-x64 --self-contained=true`.
6. Create a systemd service file in `/etc/systemd/system/<appname>.service` with `Type=notify` and `ExecStart=<path-to-executable>`.
7. Reload systemd configuration: `sudo systemctl daemon-reload`.
8. Enable and start the service: `sudo systemctl enable <appname>.service && sudo systemctl start <appname>.service`.
9. Verify status and logs: `sudo systemctl status <appname>.service` and `sudo journalctl -u <appname>.service`.

## Process Signatures

- **Process name / executable:** The process name is the application name (e.g., `myworkerapp`, `dnsserver`) published as a self-contained Linux binary. The executable is often invoked directly by systemd from a path like `/opt/myapp/myworkerapp` or `/usr/local/bin/myworkerapp` rather than via `dotnet` CLI.
- **Command-line patterns:** The process runs as the binary name with minimal arguments; typical patterns are `myworkerapp` (no arguments) or `myworkerapp --configuration=/etc/myapp/config.json`. If using the `dotnet` runtime (non-self-contained), the command is `dotnet /path/to/app.dll` or `dotnet myapp.dll`.
- **Parent process:** `systemd` (PID 1 or a systemd service manager instance).
- **Typical user:** A dedicated unprivileged user created for the service (e.g., `www-data`, `app`, `svc-myworker`, or the numeric UID) as specified in the `[Service] User=` directive.
- **Working directory:** The directory specified by `WorkingDirectory=` in the systemd unit file, typically `/opt/myapp`, `/srv/myapp`, or `/home/app/myworkerapp`. If not explicitly set, defaults to the root directory `/`.

## File System Paths

### Linux

- **Install root:** Typically `/opt/<appname>`, `/srv/<appname>`, `/usr/local/bin`, or `/home/<username>/<appname>` depending on organization policy.
- **Binaries:** The compiled executable (e.g., `/opt/myworkerapp/myworkerapp` or `/usr/local/bin/myworkerapp`). For non-self-contained deployments, the .NET runtime (via `dotnet` CLI) and the application DLL (e.g., `/opt/myworkerapp/myworkerapp.dll`).
- **Application/deploy dir:** `/opt/<appname>`, `/srv/<appname>`, or `/home/<username>/<appname>` — contains the executable, DLLs, configuration files, and supporting assets.
- **PID file:** Not used by systemd; systemd tracks the process directly. However, some applications write a PID file in `/var/run/<appname>.pid` or `/run/<appname>.pid` for compatibility or monitoring purposes.

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `DOTNET_ROOT` | Path to the .NET runtime directory; required when dotnet is not in the PATH for the service user. | `/usr/local/lib/dotnet`, `/snap/dotnet/current`, or `/opt/dotnet` |
| `ASPNETCORE_ENVIRONMENT` | Logical environment name (used even for non-ASP.NET services via convention). | `Production`, `Staging`, `Development` |
| `DOTNET_CLI_TELEMETRY_OPTOUT` | Disables .NET CLI telemetry (set to `1` in production). | `1` |
| `DOTNET_SYSTEM_GLOBALIZATION_INVARIANT` | Use invariant globalization mode to reduce memory footprint. | `1` (optional) |
| Custom app settings | Application-specific configuration variables. | Depends on the app; may include API keys, connection strings, feature flags, etc. |

## Configuration Files

- **`/etc/systemd/system/<appname>.service`** — The systemd unit file. Contains `[Unit]`, `[Service]` (with `Type=notify`, `ExecStart`, `User`, `WorkingDirectory`, `Environment` directives), and `[Install]` sections. Example name: `/etc/systemd/system/myworkerapp.service`.
- **Application config file** — Often in the application directory (e.g., `/opt/myapp/appsettings.json`, `/opt/myapp/appsettings.Production.json`) or a standard location like `/etc/myapp/config.json`. Format is typically JSON and loaded by the .NET configuration framework.
- **Environment file** (optional) — A file sourced by systemd (e.g., `/etc/myapp/myapp.env` referenced via `EnvironmentFile=` in the service file) to separate configuration from the service file.

## Log Locations

- **systemd journal:** Application logs go to the systemd journal when using `Microsoft.Extensions.Hosting.Systemd`. Access via `journalctl -u <appname>.service` or `journalctl -u <appname>.service -f` (follow mode). Journal retention is managed by systemd configuration (typically in `/etc/systemd/journald.conf`).
- **Application-written logs:** If the application writes logs directly to files (outside the hosting framework), typical locations are `/var/log/<appname>/<appname>.log`, `/var/log/<appname>/error.log`, or `/var/log/<appname>/debug.log`. Rotation is typically managed by `logrotate` with a config file in `/etc/logrotate.d/<appname>`.
- **Default output redirection:** If `StandardOutput=journal` (default with `Type=notify`) and `StandardError=journal` are set, stdout and stderr are captured by the journal.

## Service / Init Integration

The process is supervised by **systemd** via a unit file named `<appname>.service` located in `/etc/systemd/system/`. The unit file is loaded by systemd during the `sudo systemctl daemon-reload` step and referenced by the service name (without the `.service` extension) in systemctl commands (e.g., `systemctl start myworkerapp`). The `[Service]` section must include `Type=notify` to enable the .NET application (via `Microsoft.Extensions.Hosting.Systemd`) to signal systemd when the host has started and is stopping. Typical systemd directives include:

```
[Unit]
Description=My Worker Service
After=network.target

[Service]
Type=notify
ExecStart=/opt/myworkerapp/myworkerapp
WorkingDirectory=/opt/myworkerapp
User=app
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

The service is enabled for automatic startup on boot with `systemctl enable <appname>.service` and controlled with standard systemctl commands (`start`, `stop`, `restart`, `status`).

## Detection Heuristics

1. **Executable name and parent:** The process name matches the application name (self-contained binary, e.g., `myworkerapp`) or is `dotnet` (runtime-dependent), and the parent process is `systemd` (PID 1 or a cgroup-aware systemd instance).

2. **Environment variable presence:** Look for `DOTNET_ROOT` in the process environment, which is set in the systemd unit file and typically indicates a .NET application. Additionally, check for custom application environment variables or `ASPNETCORE_ENVIRONMENT`.

3. **systemd integration markers:** Check for the presence of `/etc/systemd/system/<appname>.service` with `Type=notify` in the `[Service]` section, confirming systemd lifecycle integration.

4. **Working directory:** The working directory is typically not the root (`/`) but an application-specific path like `/opt/<appname>` or `/srv/<appname>`, distinguishing it from ad-hoc console applications.

5. **Loaded assemblies:** If inspecting `/proc/[pid]/maps`, look for `Microsoft.Extensions.Hosting.Systemd` or `hostfxr` (the .NET host resolver) shared objects, confirming this is a .NET generic host process.

6. **No HTTP listeners:** Unlike ASP.NET Core applications, .NET Worker Services do not listen on HTTP ports (80, 443, or custom web ports). The process does not bind to any TCP listen socket in `netstat` or `/proc/[pid]/net/tcp`.

## Version / Variant Differences

- **.NET version compatibility:** The `Microsoft.Extensions.Hosting.Systemd` package is available in .NET Core 3.1 and later (.NET 5, 6, 7, 8, etc.). The package version number aligns with the .NET version (e.g., version 8.0.x for .NET 8).

- **Self-contained vs. runtime-dependent:** Self-contained applications (published with `--self-contained=true`) bundle the .NET runtime and run as a standalone executable. Runtime-dependent applications require `DOTNET_ROOT` to be set and invoke `dotnet` CLI to execute the DLL. Detect this by checking if the process is the app executable name directly or if it is `dotnet` with a `.dll` argument.

- **Single-file applications:** Modern .NET allows publishing as a single executable file (`dotnet publish --self-contained -p:PublishSingleFile=true`), which appears as one binary with no separate DLL files in the directory.

- **Container vs. bare-metal:** This skill covers bare-metal/VM systemd deployments only. Containerized .NET services (Docker, Podman, Kubernetes) do not use systemd integration on the host and should not match this skill.

## Sources

- [.NET Core and systemd - .NET Blog](https://devblogs.microsoft.com/dotnet/net-core-and-systemd/)
- [dotnet new worker - Scott Hanselman's Blog](https://www.hanselman.com/blog/dotnet-new-worker-windows-services-or-linux-systemd-services-in-net-core)
- [How to run a .NET Core console app as a service using Systemd on Linux](https://swimburger.net/blog/dotnet/how-to-run-a-dotnet-core-console-app-as-a-service-using-systemd-on-linux)
- [Running a .NET application as a service on Linux with Systemd](https://dev.to/maartenba/running-a-net-application-as-a-service-on-linux-with-systemd-4n6n)
- [Microsoft.Extensions.Hosting.Systemd NuGet Package](https://www.nuget.org/packages/Microsoft.Extensions.Hosting.Systemd/)
