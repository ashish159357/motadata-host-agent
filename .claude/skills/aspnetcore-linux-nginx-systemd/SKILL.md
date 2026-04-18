---
name: aspnetcore-linux-nginx-systemd
description: ASP.NET Core (Kestrel) behind nginx reverse proxy on Linux, managed by systemd; detectable via dotnet process, ASPNETCORE_ENVIRONMENT env var, /etc/systemd/system/*.service with ExecStart=/usr/bin/dotnet, and nginx reverse proxy to localhost:5000.
---

# ASP.NET Core on Linux — Kestrel behind nginx with systemd

## Overview

This is a production-standard deployment pattern where ASP.NET Core runs on Kestrel (an embedded HTTP server) behind an Nginx reverse proxy. Nginx listens on port 80 (and optionally 443 for HTTPS), forwards requests to Kestrel listening on localhost:5000, and handles SSL termination, static content caching, and request compression. A systemd service unit manages the Kestrel process lifecycle, ensuring automatic restart on failure and startup at boot.

This pattern is the recommended approach for hosting ASP.NET Core on Linux (Ubuntu, RHEL, SUSE) in production environments, offering a clear separation between the public-facing web server and the application server.

## Deployment Process

1. **Install prerequisites:**
   - Install the .NET runtime: `sudo apt-get install dotnet-runtime-X.0` (Ubuntu) or equivalent for RHEL/SUSE.
   - Install Nginx: `sudo apt-get install nginx` (Ubuntu) or `sudo yum install nginx` (RHEL).

2. **Publish the ASP.NET Core application:**
   ```bash
   dotnet publish --configuration Release
   ```
   Output is typically in `bin/Release/{TFM}/publish/`.

3. **Copy the published app to the server:**
   ```bash
   scp -r bin/Release/net8.0/publish/ user@server:/var/www/helloapp/
   ```
   Standard location is `/var/www/<appname>/`.

4. **Set ownership and permissions:**
   ```bash
   sudo chown -R www-data:www-data /var/www/helloapp/
   sudo chmod -R 755 /var/www/helloapp/
   ```

5. **Create the systemd service file:**
   ```bash
   sudo nano /etc/systemd/system/kestrel-helloapp.service
   ```
   Add the unit file content (see Configuration Files section).

6. **Enable and start the service:**
   ```bash
   sudo systemctl enable kestrel-helloapp.service
   sudo systemctl start kestrel-helloapp.service
   sudo systemctl status kestrel-helloapp.service
   ```

7. **Configure Nginx as reverse proxy:**
   Edit `/etc/nginx/sites-available/default` (Ubuntu) or `/etc/nginx.conf` (RHEL/SUSE). Add a `server` block with `proxy_pass http://127.0.0.1:5000/;` and other proxy headers (see Configuration Files section).

8. **Test and reload Nginx:**
   ```bash
   sudo nginx -t
   sudo nginx -s reload
   ```

## Process Signatures

- **Process name / executable:** `dotnet` (the .NET runtime)
- **Command-line patterns:** `/usr/bin/dotnet /var/www/<appname>/<appname>.dll`, optionally with arguments like `--port 5000`
- **Parent process:** `systemd` (PID 1)
- **Typical user:** `www-data` (on Debian/Ubuntu) or `nginx` (on some RHEL/SUSE systems)
- **Working directory:** `/var/www/<appname>/` or similar application deployment root

## File System Paths

### Linux

- **Install root:** `/var/www/<appname>/` (common convention; configurable)
- **Binaries:** `/usr/bin/dotnet` (runtime), `/var/www/<appname>/<appname>.dll` (compiled assembly)
- **Application/deploy dir:** `/var/www/<appname>/` (contains published app, appsettings.json, wwwroot, etc.)
- **PID file:** None standard; systemd manages the PID internally; retrieve via `systemctl status <service-name>` or `systemctl show -p MainPID <service-name>`
- **systemd service file:** `/etc/systemd/system/kestrel-<appname>.service` or `/etc/systemd/system/<appname>.service`
- **Nginx configuration:** `/etc/nginx/sites-available/default` (Ubuntu) or `/etc/nginx.conf` (RHEL/SUSE)
- **Nginx sites-enabled:** `/etc/nginx/sites-enabled/default` (Ubuntu, typically a symlink)

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| ASPNETCORE_ENVIRONMENT | Runtime environment (affects appsettings.{env}.json selection) | `Production`, `Staging`, `Development` |
| ASPNETCORE_URLS | Kestrel bind addresses (optional; usually left unset to use code defaults) | `http://localhost:5000` |
| DOTNET_NOLOGO | Suppresses .NET SDK logo on startup (optional) | `true` |
| DOTNET_PRINT_TELEMETRY_MESSAGE | Suppresses telemetry notice (optional) | `false` |
| ConnectionStrings__DefaultConnection | Database connection string (double underscore replaces `:` in env vars) | `Server=localhost;Database=mydb;User=sa;Password=...` |

## Configuration Files

- **`/etc/systemd/system/kestrel-<appname>.service`** — systemd unit file (INI format). Defines the service: working directory, executable path, restart policy, environment variables, user account, and logging sink. Example:

  ```ini
  [Unit]
  Description=Example .NET Web API App running on Linux

  [Service]
  WorkingDirectory=/var/www/helloapp
  ExecStart=/usr/bin/dotnet /var/www/helloapp/helloapp.dll
  Restart=always
  RestartSec=10
  KillSignal=SIGINT
  SyslogIdentifier=dotnet-helloapp
  User=www-data
  Environment=ASPNETCORE_ENVIRONMENT=Production
  Environment=DOTNET_NOLOGO=true
  TimeoutStopSec=90

  [Install]
  WantedBy=multi-user.target
  ```

- **`/etc/nginx/sites-available/default`** (Ubuntu) or **`/etc/nginx.conf`** (RHEL/SUSE) — Nginx reverse proxy configuration. Maps port 80 (and 443 for HTTPS) to localhost:5000 where Kestrel listens. Example:

  ```text
  map $http_connection $connection_upgrade {
    "~*Upgrade" $http_connection;
    default keep-alive;
  }

  server {
    listen        80;
    server_name   example.com *.example.com;
    location / {
        proxy_pass         http://127.0.0.1:5000/;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection $connection_upgrade;
        proxy_set_header   Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }
  }
  ```

- **`/var/www/<appname>/appsettings.json`** — Application configuration file (JSON). Contains logging, database connection strings, and feature flags.

- **`/var/www/<appname>/appsettings.Production.json`** — Environment-specific overrides, loaded when `ASPNETCORE_ENVIRONMENT=Production`.

## Log Locations

- **systemd journal:** Primary logging sink for the Kestrel process. View logs with:
  ```bash
  sudo journalctl -fu kestrel-<appname>.service
  ```
  Logs are written to the system journal under the `SyslogIdentifier` (e.g., `dotnet-helloapp`).

- **Application logs:** Typically written to `/var/www/<appname>/logs/` if the app configures file-based logging (via Serilog, NLog, etc.). Paths depend on appsettings.json configuration.

- **Nginx access logs:** `/var/log/nginx/access.log`
- **Nginx error logs:** `/var/log/nginx/error.log`

## Service / Init Integration

**systemd service unit:** `/etc/systemd/system/kestrel-<appname>.service` (or just `<appname>.service`)

The unit file defines:
- Service name: `kestrel-<appname>` or `<appname>`
- Start command: `ExecStart=/usr/bin/dotnet /var/www/<appname>/<appname>.dll`
- Restart behavior: `Restart=always` with `RestartSec=10` (restarts after 10 seconds on crash)
- User context: `User=www-data`
- Logging: via systemd journal; filter with `journalctl -u kestrel-<appname>.service`

**Common systemd commands:**
```bash
sudo systemctl status kestrel-<appname>.service      # View service status and PID
sudo systemctl start kestrel-<appname>.service       # Start the service
sudo systemctl stop kestrel-<appname>.service        # Stop the service
sudo systemctl restart kestrel-<appname>.service     # Restart the service
sudo systemctl enable kestrel-<appname>.service      # Enable autostart at boot
sudo systemctl disable kestrel-<appname>.service     # Disable autostart
```

## Detection Heuristics

A detection tool should look for:

1. **Process matching:** Executable name is `/usr/bin/dotnet` (or similar .NET runtime path).
2. **Command-line argument:** Second argument is a `.dll` file path matching `/var/www/*/` or `/opt/*/` patterns.
3. **Environment variable:** `ASPNETCORE_ENVIRONMENT` is set (any value).
4. **Parent process:** Parent PID is 1 (systemd) or a systemd cgroup is present.
5. **Service file presence:** `/etc/systemd/system/*.service` exists with `ExecStart=/usr/bin/dotnet` and `ExecStart=` pointing to a `.dll`.
6. **Nginx reverse proxy:** `/etc/nginx/sites-available/default`, `/etc/nginx/sites-enabled/default`, or `/etc/nginx.conf` contains `proxy_pass http://127.0.0.1:5000` (or similar localhost port), indicating this dotnet process is fronted by nginx.

**Highest-confidence detection combination:**
- Process: `/usr/bin/dotnet <path>/*.dll`
- Environment: `ASPNETCORE_ENVIRONMENT` present
- Service file: `/etc/systemd/system/*.service` with `ExecStart=/usr/bin/dotnet`
- Nginx config: `proxy_pass http://localhost:5000` or `proxy_pass http://127.0.0.1:5000`

## Version / Variant Differences

- **ASP.NET Core versions 3.1+** (including LTS versions like 6.0, 8.0, 10.0): Configuration and systemd integration are consistent. The main difference is the .NET TFM in publish output (e.g., `net6.0` vs. `net8.0`) and the runtime package names.
- **Default Kestrel port:** Always `5000` in the official Microsoft examples; some custom deployments may use `5001` or other ports, but `5000` is the standard.
- **User account:** `www-data` on Debian/Ubuntu; on RHEL/SUSE, may be `nginx`, `apache`, or a custom user. The systemd service file always specifies the user in the `User=` directive.
- **Nginx installation source:** Official nginx packages from `nginx.org` (Ubuntu, RHEL, SUSE) vs. distro-provided packages (often older). Official packages are recommended.
- **Framework-dependent vs. self-contained deployment:** Framework-dependent (FDD) requires the .NET runtime pre-installed; self-contained (SCD) bundles the runtime. Both use the same systemd integration approach.

## Sources

- [Host ASP.NET Core on Linux with Nginx | Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/linux-nginx?view=aspnetcore-10.0)
- [Configure the ASP.NET Core application to start automatically - ASP.NET Core | Microsoft Learn](https://learn.microsoft.com/en-us/troubleshoot/developer/webapps/aspnetcore/practice-troubleshoot-linux/2-3-configure-aspnet-core-application-start-automatically)
