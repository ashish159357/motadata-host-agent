---
name: aspnetcore-linux-nginx-systemd
description: Kestrel-hosted ASP.NET Core application running behind nginx reverse proxy, supervised by systemd; detected by dotnet process with ASPNETCORE_ENVIRONMENT env var, systemd service file at /etc/systemd/system/kestrel-*.service, and nginx configuration at /etc/nginx/sites-available/* or /etc/nginx/conf.d/*
---

# ASP.NET Core on Linux — Kestrel behind nginx with systemd

## Overview

This is the standard production deployment pattern for ASP.NET Core applications on Linux (Ubuntu, RHEL, SUSE). Kestrel is a lightweight, cross-platform HTTP server embedded in the .NET runtime; it runs the ASP.NET Core application directly. nginx acts as a reverse proxy on the same host, terminating HTTP connections, handling SSL/TLS termination, serving static files, and forwarding dynamic requests to Kestrel. systemd supervises the Kestrel process, ensuring automatic restart on failure and clean integration with the Linux boot process. This architecture is recommended by Microsoft for production Linux deployments and is widely adopted because it provides security isolation, performance, and operational simplicity.

## Deployment Process

1. **Prepare the application environment:**
   ```bash
   mkdir -p /var/www/appname
   chown www-data:www-data /var/www/appname
   ```

2. **Deploy the ASP.NET Core application:**
   ```bash
   # Copy published .NET assemblies and supporting files to the application directory
   cp -r /path/to/published/app/* /var/www/appname/
   chown -R www-data:www-data /var/www/appname
   ```

3. **Create the systemd service file** at `/etc/systemd/system/kestrel-appname.service`:
   ```
   [Unit]
   Description=ASP.NET Core App - AppName
   After=network.target

   [Service]
   Type=notify
   WorkingDirectory=/var/www/appname
   ExecStart=/usr/bin/dotnet /var/www/appname/AppName.dll
   Restart=always
   RestartSec=10
   KillSignal=SIGINT
   SyslogIdentifier=kestrel-appname
   User=www-data
   Environment=ASPNETCORE_ENVIRONMENT=Production
   Environment=ASPNETCORE_URLS=http://127.0.0.1:5000

   [Install]
   WantedBy=multi-user.target
   ```

4. **Enable and start the systemd service:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable kestrel-appname.service
   sudo systemctl start kestrel-appname.service
   ```

5. **Configure nginx** at `/etc/nginx/sites-available/appname`:
   ```
   server {
       listen 80;
       server_name example.com www.example.com;

       location / {
           proxy_pass http://127.0.0.1:5000;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection keep-alive;
           proxy_set_header Host $host;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           proxy_buffering off;
           proxy_request_buffering off;
       }
   }
   ```

6. **Enable the nginx configuration and test:**
   ```bash
   sudo ln -s /etc/nginx/sites-available/appname /etc/nginx/sites-enabled/appname
   sudo nginx -t
   sudo systemctl restart nginx
   ```

## Process Signatures

- **Process name / executable:** `dotnet`
- **Command-line patterns:** `/usr/bin/dotnet /var/www/appname/AppName.dll`, `/usr/local/share/dotnet/dotnet /var/www/appname/*.dll`, `dotnet run`
- **Parent process:** `systemd`
- **Typical user:** `www-data` (Debian/Ubuntu) or `aspnet` (RHEL/CentOS/SUSE)
- **Working directory:** `/var/www/appname` (or custom path specified in systemd WorkingDirectory)

## File System Paths

### Linux

- **Install root:** Application directory specified in systemd WorkingDirectory, typically `/var/www/appname`
- **Binaries:** `/usr/bin/dotnet` (primary), `/usr/local/share/dotnet/dotnet` (alternative), `/snap/dotnet/current/dotnet` (snap package)
- **Application/deploy dir:** `/var/www/appname` (contains *.dll, *.json, and other published artifacts)
- **PID file:** N/A — systemd manages the process lifecycle; no explicit PID file required
- **systemd service file:** `/etc/systemd/system/kestrel-*.service`
- **nginx config:** `/etc/nginx/sites-available/appname`, `/etc/nginx/sites-enabled/appname`, or `/etc/nginx/conf.d/appname.conf`
- **nginx default config:** `/etc/nginx/nginx.conf`

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| `ASPNETCORE_ENVIRONMENT` | Application environment (Production, Development, Staging) | `Production` |
| `ASPNETCORE_URLS` | Kestrel binding address and port | `http://127.0.0.1:5000` |
| `DOTNET_NOLOGO` | Suppress .NET startup logo in logs | `true` |
| `DOTNET_SYSTEM_GLOBALIZATION_INVARIANT` | Disable globalization for performance | `false` (or `true` for reduced size) |
| `ConnectionStrings__DefaultConnection` | Database connection string (double underscore for colon) | database connection string |
| `ASPNETCORE_HTTPS_PORT` | HTTPS port if configured | `5001` |

## Configuration Files

- **`/etc/systemd/system/kestrel-appname.service`** — systemd unit file defining the Kestrel service: working directory, executable, restart policy, user, environment variables
- **`/etc/nginx/sites-available/appname`** — nginx reverse proxy configuration with upstream address (typically http://127.0.0.1:5000), proxy headers, SSL/TLS settings
- **`/var/www/appname/appsettings.json`** — ASP.NET Core configuration file: logging, connection strings, feature flags
- **`/var/www/appname/appsettings.Production.json`** — Environment-specific overrides loaded when ASPNETCORE_ENVIRONMENT=Production

## Log Locations

- **systemd journal:** Logs captured by systemd under SyslogIdentifier; query with `journalctl -u kestrel-appname.service` or `journalctl -u kestrel-appname`
- **Application logs:** Typically written to stdout and captured by systemd journal; custom file logging depends on application's logging configuration in `appsettings.json`
- **nginx access logs:** `/var/log/nginx/access.log`
- **nginx error logs:** `/var/log/nginx/error.log`

## Service / Init Integration

The Kestrel ASP.NET Core application is supervised by systemd via a service unit file named `/etc/systemd/system/kestrel-appname.service`. The service is registered with `systemctl enable kestrel-appname.service` (to start on boot) and managed with `systemctl start`, `systemctl stop`, and `systemctl status` commands. The systemd unit specifies `Type=notify` (or `Type=simple`), `Restart=always` for automatic recovery, and `User=www-data` for process permissions. nginx is managed separately as `/etc/systemd/system/nginx.service` or via the package manager's init integration.

## Detection Heuristics

1. **Process with name `dotnet` whose parent is systemd:** Indicates a systemd-supervised .NET application.
2. **Environment variable `ASPNETCORE_ENVIRONMENT` present on the dotnet process:** Strongly signals an ASP.NET Core application (not standalone .NET).
3. **Working directory `/var/www/*` with `*.dll` and `appsettings.json` present:** Application deployment location.
4. **Kestrel listening on localhost (127.0.0.1) port 5000 or 5001:** Default Kestrel ports; detectable via `/proc/[pid]/net/tcp` or netstat.
5. **nginx process listening on 0.0.0.0 port 80 or 443 with proxy_pass directive in config pointing to 127.0.0.1:5000:** Confirms the reverse proxy architecture.
6. **systemd service unit file `/etc/systemd/system/kestrel-*.service` with ExecStart pointing to dotnet and a .dll file:** Definitive confirmation of systemd supervision.
7. **Combination: dotnet process, ASPNETCORE_ENVIRONMENT env var, working directory in `/var/www/`, parent systemd, AND nginx config with proxy_pass to localhost:5000:** Near-certain identification.

## Version / Variant Differences

- **ASP.NET Core 3.1 through 10.0+:** All use the same Kestrel + systemd pattern; nginx configuration syntax unchanged.
- **.NET Framework vs. .NET Core/5+:** This skill applies only to .NET 5+ and .NET Core 2.1+; .NET Framework requires Windows or Mono.
- **User account:** May be `www-data` (Debian/Ubuntu), `aspnet` or `app` (RHEL/CentOS), or a custom account; always check systemd service file User directive.
- **Kestrel port:** Typically 5000 (HTTP) or 5001 (HTTPS); custom ports set in ASPNETCORE_URLS environment variable or appsettings.json.
- **Application directory:** May be `/var/www/appname`, `/opt/appname`, or custom path; always check systemd WorkingDirectory.
- **SSL/TLS termination:** Can be handled by nginx (most common) or Kestrel directly if ASPNETCORE_HTTPS_PORT is configured.

## Sources

- [Host ASP.NET Core on Linux with Nginx - Microsoft Learn](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/linux-nginx?view=aspnetcore-10.0)
- [Deploy ASP.NET Core Application on Linux with Nginx - Code Maze](https://code-maze.com/deploy-aspnetcore-linux-nginx/)
- [Host ASP.NET Core on Linux with Nginx - GitHub (dotnet/AspNetCore.Docs)](https://github.com/dotnet/AspNetCore.Docs/blob/main/aspnetcore/host-and-deploy/linux-nginx.md)
