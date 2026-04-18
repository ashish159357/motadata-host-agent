---
name: dotnet-selfcontained-linux-systemd
description: .NET self-contained single-file application on Linux with systemd supervision—native ELF binary executable containing runtime and dependencies, detectable by DOTNET_BUNDLE_EXTRACT_BASE_DIR env var, self-contained systemd unit in /etc/systemd/system/*.service with Type=notify, and ELF binary name matching published app name.
---

# .NET Self-Contained (Single-File) on Linux — systemd Service

## Overview
Self-contained .NET deployments publish all managed code and the .NET runtime into a single native Linux ELF executable that ships without requiring a system-wide .NET runtime installation. The app runs as a systemd service with Type=notify integration for clean lifecycle management and Journald logging. This variant is chosen by teams seeking minimal dependency footprint, simplified deployment (one file to copy), and full ownership of the runtime version independent of host updates.

## Deployment Process

1. **Publish as self-contained single file:**
   ```bash
   dotnet publish -c Release -r linux-x64 --self-contained true -p:PublishSingleFile=true -o ./publish
   ```

2. **Copy the executable to target machine** (e.g. `/opt/myapp/myapp`):
   ```bash
   scp publish/myapp user@host:/opt/myapp/
   chmod +x /opt/myapp/myapp
   ```

3. **Create systemd unit file** at `/etc/systemd/system/myapp.service`:
   ```ini
   [Unit]
   Description=My .NET App
   After=network.target

   [Service]
   Type=notify
   ExecStart=/opt/myapp/myapp
   WorkingDirectory=/opt/myapp
   User=myapp
   Restart=on-failure
   RestartSec=10
   Environment="DOTNET_BUNDLE_EXTRACT_BASE_DIR=%h/.net"

   [Install]
   WantedBy=multi-user.target
   ```

4. **Enable and start the service:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable myapp.service
   sudo systemctl start myapp.service
   ```

## Process Signatures
- **Process name / executable:** Application-specific binary name (e.g. `myapp`, `api-server`); always an ELF executable with no `.dll` or `.exe` extension on Linux.
- **Command-line patterns:** Binary path alone with no trailing arguments (e.g. `/opt/myapp/myapp`) or with config/environment overrides; often invoked by systemd with no visible arguments beyond the path.
- **Parent process:** `systemd` (PID 1 or systemd session manager for user services).
- **Typical user:** Service account (e.g. `myapp`, `dotnet`, `www-data`); rarely `root`.
- **Working directory:** Application directory specified in `WorkingDirectory=` directive (e.g. `/opt/myapp`, `/srv/app`).

## File System Paths

### Linux
- **Install root:** `/opt/<appname>`, `/srv/<appname>`, or `/usr/local/<appname>`.
- **Binaries:** `/opt/<appname>/<appname>` (the self-contained ELF executable); no `/bin` subdirectory.
- **Application/deploy dir:** Same as binary location; config files, appsettings.json, plugins often co-located.
- **PID file:** No PID file; systemd manages the process lifecycle via cgroups. Inspect with `systemctl status myapp.service`.
- **Runtime extraction:** Files extracted at runtime to `$HOME/.net/` (typically `/home/<user>/.net/`) as set by `DOTNET_BUNDLE_EXTRACT_BASE_DIR=%h/.net` in the service unit, or to `%TEMP%/.net/` if undefined (not recommended for systemd).

## Environment Variables
| Variable | Purpose | Typical value |
|---|---|---|
| `DOTNET_BUNDLE_EXTRACT_BASE_DIR` | Directory where self-contained app extracts managed DLLs and runtime libraries at startup | `%h/.net` (systemd expands `%h` to service user's home) |
| `ASPNETCORE_ENVIRONMENT` | (For ASP.NET Core apps) Deployment environment name | `Production`, `Development` |
| `ASPNETCORE_URLS` | (For ASP.NET Core apps) HTTP binding addresses | `http://localhost:5000` |
| `DOTNET_EnableDiagnostics` | Enable runtime diagnostics | `0` (disable for security) or `1` |
| `DOTNET_CLI_TELEMETRY_OPTOUT` | Opt out of .NET CLI telemetry | `1` |

## Configuration Files
- **`/etc/systemd/system/<appname>.service`** — systemd unit file defining service metadata, Type=notify, ExecStart path, user, working directory, environment variables, and restart policy.
- **`<appdir>/appsettings.json`** — ASP.NET Core app configuration (if applicable); loaded from working directory.
- **`<appdir>/appsettings.Production.json`** — Environment-specific settings (if applicable).
- **Custom config files** — Any YAML, XML, or JSON files referenced by the application; typically in the same directory as the executable or in `/etc/<appname>/`.

## Log Locations
- **Journald (systemd journal):** All logs written to `stdout`/`stderr` are captured by systemd and indexed in Journald. View logs with:
  ```bash
  journalctl -u myapp.service -f
  ```
- **Application-specific log files** (if configured): Vary by app; check `appsettings.json` for Serilog or other logging framework configuration. Often `/var/log/<appname>/` or `<appdir>/logs/`.

## Service / Init Integration
**systemd unit file:** `/etc/systemd/system/<appname>.service`

The unit **must** set `Type=notify` to enable the app to signal systemd when it is ready. The .NET application integrates via the `Microsoft.Extensions.Hosting.Systemd` NuGet package (call `UseSystemd()` in Program.cs), which:
- Notifies systemd when the host has started (SD_NOTIFY).
- Routes all logging through systemd's Journald.
- Gracefully handles systemd lifecycle signals (SIGTERM).

Service names and unit file paths follow the pattern:
- Unit identifier: `<appname>.service` (e.g. `myapp.service`, `api-server.service`)
- Full path: `/etc/systemd/system/<appname>.service`
- Enable with: `systemctl enable <appname>.service`
- Start with: `systemctl start <appname>.service`
- Check status: `systemctl status <appname>.service` or `journalctl -u <appname>.service`

## Detection Heuristics

1. **Executable type:** Process command is a native Linux ELF binary (file type check: `file /proc/<pid>/exe` returns `ELF 64-bit LSB executable`), not a script or managed wrapper.

2. **Environment variable:** Presence of `DOTNET_BUNDLE_EXTRACT_BASE_DIR` environment variable in the process environment (`/proc/<pid>/environ`), or presence of extracted .NET runtime files under `$HOME/.net/` for the service user.

3. **Systemd parent and unit file:** Parent process is `systemd` (or systemd session manager), and `/etc/systemd/system/<procname>.service` exists with `Type=notify` in the `[Service]` section.

4. **Working directory:** Application working directory contains `appsettings.json` or other .NET configuration files alongside the single-file ELF executable.

5. **Shared library patterns:** The process maps contain isolated .NET runtime libraries (e.g. `libcoreclr.so`, `libmono.so`, `libcrypto.so`) extracted and loaded from `$HOME/.net/` rather than from system paths, indicating bundled dependencies.

6. **Process name:** Absence of `.dll` or `.exe` extensions; binary name is the app name (e.g. `myapp`, not `myapp.exe` or `myapp.dll`).

## Version / Variant Differences

- **.NET 5 and later:** Self-contained single-file publishing is fully supported. Trimming and compression options available in .NET 6+.
- **.NET 3.x / .NET Core 3.x:** Self-contained deployment supported; single-file option introduced in `.NET Core 3.0`. systemd integration available via `Microsoft.Extensions.Hosting.Systemd` NuGet package.
- **Runtime extraction:** In .NET 5+, managed DLLs are extracted to memory by default; only native binaries may be extracted to disk (unless `IncludeNativeLibrariesForSelfExtract=true` is set during publish).
- **Linux runtime identifiers:** Common RIDs are `linux-x64`, `linux-arm64`, `linux-musl-x64` (Alpine). Detector must account for architecture-specific binaries deployed on matching architectures only.
- **Trimming:** .NET 6+ supports trimming to reduce single-file size; presence of trimmed executables will show smaller `.so` libraries and reduced binary footprint.

## Sources
- [Create a single file for application deployment - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/deploying/single-file/overview)
- [.NET Core and systemd - .NET Blog](https://devblogs.microsoft.com/dotnet/net-core-and-systemd/)
- [Self-Contained Linux Applications in .NET Core](https://github.com/dotnet/core/blob/main/Documentation/self-contained-linux-apps.md)
- [.NET application publishing overview - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/deploying/)
- [dotnet publish command - .NET CLI | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-publish)
