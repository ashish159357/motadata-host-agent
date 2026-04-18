---
name: aspnetcore-linux-systemd
description: ASP.NET Core application running on Linux with Kestrel web server managed by systemd service; detect by process executable `dotnet`, ASPNETCORE_ENVIRONMENT env var in systemd unit file, and .service file in /etc/systemd/system/ or working directory containing *.dll or appsettings.json.
---

# ASP.NET Core on Linux — Kestrel via systemd

## Overview

ASP.NET Core applications using the built-in Kestrel web server running directly on Linux systems and managed by systemd service units. This is a common production deployment pattern for .NET applications on Linux distributions (Ubuntu, RHEL, SUSE) as an alternative to running behind a reverse proxy. Kestrel is the cross-platform web server included in ASP.NET Core by default and is fully supported for direct internet-facing or internal network deployment when managed by systemd.

## Deployment Process

1. Publish the ASP.NET Core application in Release configuration:
   ```bash
   dotnet publish --configuration Release
   ```

2. Copy the published application to a deployment directory on the Linux host (typically `/var/www/<appname>`):
   ```bash
   scp -r bin/Release/net8.0/publish/* user@host:/var/www/helloapp/
   ```

3. Create a systemd service unit file at `/etc/systemd/system/<servicename>.service`:
   ```bash
   sudo nano /etc/systemd/system/kestrel-helloapp.service
   ```

4. Add service configuration with at minimum:
   - `ExecStart` pointing to `dotnet` and the application DLL
   - `WorkingDirectory` pointing to the application directory
   - `User` (typically `www-data` or `app` user)
   - `Environment` variables including `ASPNETCORE_ENVIRONMENT=Production`
   - `Restart=always` for automatic recovery

5. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable kestrel-helloapp.service
   sudo systemctl start kestrel-helloapp.service
   ```

6. Verify the service is running:
   ```bash
   sudo systemctl status kestrel-helloapp.service
   ```

## Process Signatures

- **Process name / executable:** `dotnet` (the .NET runtime executable)
- **Command-line patterns:** `dotnet /var/www/<appname>/<appname>.dll`, `dotnet /path/to/*.dll` (framework-dependent deployment); or self-contained executable name for self-contained deployments
- **Parent process:** `systemd` (PID 1 or systemd instance)
- **Typical user:** `www-data` (Debian/Ubuntu), `app` (custom service user), or other unprivileged service account
- **Working directory:** `/var/www/<appname>` or `/opt/<appname>` (the directory containing the application DLL and configuration files)

## File System Paths

### Linux

- **Install root:** `/var/www/<appname>`, `/opt/<appname>`, or custom deployment directory specified by operator
- **Binaries:** `/usr/bin/dotnet` (the .NET runtime) or `/usr/local/bin/dotnet` (local installation)
- **Application/deploy dir:** `/var/www/<appname>`, `/opt/<appname>` (contains `*.dll`, `appsettings.json`, `appsettings.{Environment}.json`, `web.config` if present)
- **PID file:** Not applicable; systemd manages the process directly. Process ID tracked by systemd and queryable via `systemctl status <service>`

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| ASPNETCORE_ENVIRONMENT | Specifies the runtime environment (determines which appsettings file is loaded) | `Production`, `Staging`, or `Development` |
| ASPNETCORE_URLS | Configures Kestrel listen endpoints (semicolon-separated list) | `http://localhost:5000;https://localhost:5001` or `http://127.0.0.1:5000` |
| ASPNETCORE_CONTENTROOT | Root directory for the application's content files | `/var/www/helloapp` (defaults to application directory) |
| ASPNETCORE_WEBROOT | Subdirectory for static web files | `wwwroot` (relative to content root) |
| DOTNET_NOLOGO | Suppresses the .NET logo and telemetry message on startup | `true` |
| DOTNET_CLI_TELEMETRY_OPTOUT | Disables telemetry collection | `1` |
| ConnectionStrings__DefaultConnection | Database connection string (double underscore replaces colon) | `Server=localhost;Database=appdb;User Id=sa;Password=...` |

## Configuration Files

- **`/etc/systemd/system/<servicename>.service`** — systemd unit file defining service startup, environment variables, restart behavior, and resource constraints. Example filename: `kestrel-helloapp.service`
- **`/var/www/<appname>/appsettings.json`** — Default configuration file for all environments
- **`/var/www/<appname>/appsettings.Production.json`** — Environment-specific configuration loaded when `ASPNETCORE_ENVIRONMENT=Production`
- **`/var/www/<appname>/appsettings.{Environment}.json`** — Environment-specific overrides
- **`/var/www/<appname>/<appname>.dll`** — The compiled application assembly (for framework-dependent deployments)
- **`/var/www/<appname>/<appname>.runtimeconfig.json`** — Runtime configuration for framework-dependent deployments
- **`/var/www/<appname>/<appname>.deps.json`** — Dependency manifest for the application

## Log Locations

- **systemd journal:** Logs are captured by systemd and queryable via `journalctl`:
  ```bash
  journalctl -u kestrel-helloapp.service -f
  journalctl -u kestrel-helloapp.service --since today
  ```
- **Application logs:** If the ASP.NET Core app logs to files, they are typically in `/var/www/<appname>/logs/` or configured via `appsettings.json` logging section
- **stdout/stderr:** Captured by systemd journal; not written to traditional log files unless explicitly configured in the application

## Service / Init Integration

**systemd unit file** located at `/etc/systemd/system/<servicename>.service` (e.g., `/etc/systemd/system/kestrel-helloapp.service`).

Example unit file structure:
```ini
[Unit]
Description=Example ASP.NET Core App running on Linux
After=network.target

[Service]
Type=notify
ExecStart=/usr/bin/dotnet /var/www/helloapp/helloapp.dll
WorkingDirectory=/var/www/helloapp
Restart=always
RestartSec=10
KillSignal=SIGINT
SyslogIdentifier=kestrel-helloapp
User=www-data
Environment=ASPNETCORE_ENVIRONMENT=Production
Environment=ASPNETCORE_URLS=http://127.0.0.1:5000
Environment=DOTNET_NOLOGO=true

[Install]
WantedBy=multi-user.target
```

**Unit service name:** Matches the filename, e.g., `kestrel-helloapp.service`. Managed via `systemctl start|stop|enable|disable|status kestrel-helloapp.service`.

## Detection Heuristics

- **Process executable:** Check `/proc/[pid]/exe` symlink points to `dotnet` executable in `/usr/bin/`, `/usr/local/bin/`, or `.dotnet/dotnet`
- **Command-line check:** Read `/proc/[pid]/cmdline` for `.dll` argument (e.g., `helloapp.dll`, `myapp.dll`)
- **Environment variable:** Check `/proc/[pid]/environ` for `ASPNETCORE_ENVIRONMENT` or `ASPNETCORE_URLS`
- **Working directory marker files:** Read `/proc/[pid]/cwd` and check for presence of `appsettings.json`, `appsettings.Production.json`, `*.csproj`, or `*.dll` files
- **Systemd integration:** Query systemd unit file `/etc/systemd/system/*.service` with pattern matching for `ExecStart.*dotnet.*\.dll` and `ASPNETCORE_ENVIRONMENT`
- **Service status check:** Use `systemctl is-active <servicename>` to confirm service is managed by systemd

## Version / Variant Differences

- **Framework-dependent deployment:** Application runs via `dotnet <appname>.dll` command; requires .NET runtime on host. Process shows as `dotnet` in ps/top.
- **Self-contained deployment:** Application is published with runtime included; executable is application name (e.g., `helloapp`). Process shows as application name, not `dotnet`.
- **ASP.NET Core 3.1+:** All modern versions support systemd socket activation (optional) and systemd notification protocol (optional Type=notify).
- **Environment-specific configs:** Apps may load `appsettings.<Environment>.json` based on `ASPNETCORE_ENVIRONMENT` variable; case-sensitive on Linux.

## Sources

- [Host ASP.NET Core on Linux with Nginx — Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/linux-nginx?view=aspnetcore-10.0)
- [Kestrel web server in ASP.NET Core — Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/fundamentals/servers/kestrel?view=aspnetcore-10.0)
