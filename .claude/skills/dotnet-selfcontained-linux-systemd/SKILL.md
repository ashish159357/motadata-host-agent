---
name: dotnet-selfcontained-linux-systemd
description: .NET self-contained single-file deployment on Linux managed by systemd; detect via exe extensionless binary in /opt or /usr/local, systemd service unit in /etc/systemd/system/*.service, and environment variable DOTNET_BUNDLE_EXTRACT_BASE_DIR set to %h/.net.
---

# .NET Self-Contained Deployment on Linux — Systemd Service

## Overview

.NET self-contained deployment (SCD) is a publishing mode where the application binary includes the .NET runtime, framework libraries, and all dependencies. When combined with single-file publishing, all files are bundled into a single extensionless executable. On Linux, systemd manages the process lifecycle, allowing the application to run as a daemon service. This pattern is popular in production environments where operators want complete control over .NET version isolation, no runtime pre-installation requirement, and native Linux service integration.

## Deployment Process

1. **Publish the application as self-contained single-file:**
   ```
   dotnet publish -c Release -r linux-x64 --self-contained true -p:PublishSingleFile=true -p:GenerateRuntimeConfigurationFiles=true -o artifacts
   ```
   This generates an extensionless binary (e.g., `myapp`) in the `artifacts` directory for the target platform.

2. **Create installation directory and copy binary:**
   ```
   sudo mkdir -p /opt/myapp
   sudo cp artifacts/myapp /opt/myapp/
   sudo chmod 0755 /opt/myapp/myapp
   sudo chown appuser:appgroup /opt/myapp/myapp
   ```

3. **Create systemd service unit file:**
   Place a `.service` file in `/etc/systemd/system/` with the following structure:
   ```
   [Unit]
   Description=My .NET Application
   After=network.target

   [Service]
   Type=notify
   ExecStart=/opt/myapp/myapp
   WorkingDirectory=/opt/myapp
   User=appuser
   Group=appgroup
   Environment="DOTNET_BUNDLE_EXTRACT_BASE_DIR=%h/.net"
   Restart=on-failure
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```

4. **Reload systemd and start the service:**
   ```
   sudo systemctl daemon-reload
   sudo systemctl start myapp.service
   sudo systemctl enable myapp.service
   ```

5. **Verify status:**
   ```
   systemctl status myapp.service
   journalctl -u myapp.service -f
   ```

## Process Signatures

- **Process name / executable:** Extensionless binary name as specified in systemd unit (e.g., `myapp`)
- **Command-line patterns:** The binary path from `ExecStart=` directive; no arguments unless explicitly configured
- **Parent process:** `systemd` (PID 1 or cgroup-scoped systemd manager)
- **Typical user:** Custom unprivileged user (e.g., `appuser`), configured via `User=` in service unit
- **Working directory:** Configured via `WorkingDirectory=` in service unit (e.g., `/opt/myapp`)

## File System Paths

### Linux

- **Install root:** `/opt/<appname>` (convention; can be any path specified in `ExecStart`)
- **Binaries:** `/opt/<appname>/<appname>` (the extensionless executable from publish output)
- **Application/deploy dir:** `/opt/<appname>` (same as install root; configuration files or plugins may be placed here)
- **Runtime extraction dir:** `$HOME/.net/` (default, or custom path set via `DOTNET_BUNDLE_EXTRACT_BASE_DIR`; systemd expands `%h` to user's home directory)
- **Service unit file:** `/etc/systemd/system/<appname>.service`
- **PID file:** None (systemd manages PID internally); process PID available via `systemctl show -p MainPID <appname>.service`

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `DOTNET_BUNDLE_EXTRACT_BASE_DIR` | Base directory for extraction of bundled files from single-file binary; required for systemd to avoid $HOME undefined error | `%h/.net` (systemd will expand %h to user home) |
| `ASPNETCORE_URLS` | (ASP.NET Core apps) Addresses and ports to bind to | `http://0.0.0.0:5000` |
| `ASPNETCORE_ENVIRONMENT` | (ASP.NET Core apps) Environment name (Development, Staging, Production) | `Production` |
| `DOTNET_CLI_TELEMETRY_OPTOUT` | Disable .NET SDK telemetry (if enabled) | `true` |

## Configuration Files

- **Service unit file** — `/etc/systemd/system/<appname>.service`; defines how systemd starts, stops, restarts, and monitors the .NET application. The `Type=notify` setting enables integration with `Microsoft.Extensions.Hosting.Systemd` for graceful shutdown signalling.
- **Application configuration** — Any `appsettings.json`, `appsettings.Production.json`, or custom config files should be placed in the `WorkingDirectory` (e.g., `/opt/myapp/appsettings.json`); the application reads them relative to `AppContext.BaseDirectory` at runtime.

## Log Locations

- **systemd journal:** All stdout and stderr from the process are captured by systemd and stored in the journal. View logs using `journalctl -u <appname>.service` or `journalctl -u <appname>.service -f` for live tail.
- **Application logs:** If the application writes to files, logs are typically written to a directory configured within the application (e.g., via `appsettings.json`). Ensure the service user has write permissions to that directory.

## Service / Init Integration

**systemd service unit file:** `/etc/systemd/system/<appname>.service`

Key systemd directives for .NET self-contained deployments:

- `Type=notify` — Tells systemd to wait for a "ready" notification from the application. Requires the NuGet package `Microsoft.Extensions.Hosting.Systemd` and a call to `.UseSystemd()` in `Program.cs`. Without this, systemd considers the service started as soon as the process forks.
- `ExecStart=` — Absolute path to the extensionless binary; must be executable.
- `WorkingDirectory=` — Sets the process working directory; used to locate relative config files.
- `User=` and `Group=` — Unprivileged user and group to run the service.
- `Restart=on-failure` — Automatically restart the service if it exits with a non-zero status.
- `RestartSec=5` — Wait 5 seconds before restarting.
- `Environment=` — Set environment variables (e.g., `DOTNET_BUNDLE_EXTRACT_BASE_DIR=%h/.net`).

**Enable auto-start at boot:**
```
sudo systemctl enable <appname>.service
```

## Detection Heuristics

1. **Process executable name:** Look for an extensionless binary name running under systemd (e.g., a binary at `/opt/myapp/myapp` with no `.exe` or `.dll` extension).

2. **Environment variable:** Check for `DOTNET_BUNDLE_EXTRACT_BASE_DIR` set to `%h/.net` or a similar bundle extraction path in the process environment (`/proc/[pid]/environ`).

3. **Systemd unit file:** Search `/etc/systemd/system/` for `.service` files with `Type=notify` in the `[Service]` section and an `ExecStart=` pointing to an executable in `/opt/`, `/usr/local/bin/`, or similar application directories.

4. **Binary content:** The executable is a self-contained .NET binary. Its initial bytes are not a standard ELF shebang; it contains embedded .NET runtime and assemblies. Running `file <binary>` will return "ELF 64-bit LSB executable" or similar, but the binary is significantly larger (tens or hundreds of MB) than a typical apphost stub.

5. **Working directory:** The process runs with a `WorkingDirectory` that often contains `appsettings.json`, `appsettings.Production.json`, or other application-specific config files.

## Version / Variant Differences

- **.NET version:** Self-contained deployments pin a specific .NET version (e.g., .NET 6, 7, 8, 9) at publish time. Operators cannot use a newer runtime already installed on the host; the bundled version is used exclusively. Determine the version by examining the binary's embedded metadata or the project file used during publish.

- **Single-file vs. folder publish:** This skill covers single-file deployments only. Folder deployments place the binary alongside `.deps.json`, `.runtimeconfig.json`, and other support files; they appear as multiple files in the directory. Single-file deployments bundle all of these into one binary.

- **Trimming and ReadyToRun:** Applications published with `-p:PublishTrimmed=true` or `-p:PublishReadyToRun=true` are still self-contained and systemd-managed, but exhibit different startup and memory profiles. No process-level detection changes; the binary is larger (ReadyToRun) or smaller (Trimmed).

- **systemd integration package:** Applications that call `.UseSystemd()` from `Microsoft.Extensions.Hosting.Systemd` register with systemd as a "notify" service and log to the journal. Those without the package still run under systemd but with `Type=simple` and may write logs separately. Detection should handle both cases.

## Sources

- [Create a single file for application deployment - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/deploying/single-file/overview)
- [.NET application publishing overview - .NET | Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/core/deploying/)
- [.NET Core and systemd - .NET Blog](https://devblogs.microsoft.com/dotnet/net-core-and-systemd/)
- [Running .NET Applications as a Systemd Service on Linux - Maarten Balliauw Blog](https://blog.maartenballiauw.be/posts/2021-05-25-running-a-net-application-as-a-service-on-linux-with-systemd/)
- [How to run a .NET Core console app as a service using Systemd on Linux (RHEL) - Swimburger](https://swimburger.net/blog/dotnet/how-to-run-a-dotnet-core-console-app-as-a-service-using-systemd-on-linux)
