---
name: aspnetcore-linux-apache-systemd
description: ASP.NET Core running on Kestrel behind Apache mod_proxy reverse proxy on Linux with systemd init; detect by dotnet process listening on localhost:5000, ASPNETCORE_ENVIRONMENT env var, and /etc/systemd/system/kestrel-*.service unit file.
---

# ASP.NET Core on Linux — Kestrel behind Apache with systemd

## Overview

This deployment pattern runs an ASP.NET Core application using the built-in Kestrel web server on a local high-numbered port (typically 5000 or 5001), with Apache httpd acting as a reverse proxy on the public-facing port 80/443. Apache uses `mod_proxy` and `mod_proxy_http` to forward HTTP requests to Kestrel and rewrite response headers. systemd manages the lifecycle of the Kestrel process as a persistent service. This architecture is widely used in production on Ubuntu, CentOS, and RHEL servers because it separates concerns: Apache handles TLS termination, caching, and static content while Kestrel runs the application backend with minimal overhead.

## Deployment Process

1. **Install the ASP.NET Core runtime:**
   ```bash
   sudo apt-get update && sudo apt-get install -y aspnetcore-runtime-X.X
   ```
   (where X.X is the target version, e.g., 7.0, 8.0)

2. **Publish the application:**
   ```bash
   dotnet publish -c Release -o /var/www/myapp
   ```

3. **Create application directory and set permissions:**
   ```bash
   sudo mkdir -p /var/www/myapp
   sudo chown -R www-data:www-data /var/www/myapp
   sudo chmod -R 755 /var/www/myapp
   ```

4. **Create systemd unit file** at `/etc/systemd/system/kestrel-myapp.service`:
   ```ini
   [Unit]
   Description=ASP.NET Core Web Application
   After=network.target

   [Service]
   Type=notify
   User=www-data
   WorkingDirectory=/var/www/myapp
   ExecStart=/usr/bin/dotnet /var/www/myapp/MyApp.dll
   Restart=always
   RestartSec=10
   KillSignal=SIGINT
   SyslogIdentifier=dotnet-myapp
   Environment=ASPNETCORE_ENVIRONMENT=Production

   [Install]
   WantedBy=multi-user.target
   ```

5. **Enable and start the systemd service:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable kestrel-myapp.service
   sudo systemctl start kestrel-myapp.service
   ```

6. **Enable Apache proxy modules:**
   ```bash
   sudo a2enmod proxy proxy_http headers rewrite
   sudo systemctl restart apache2
   ```

7. **Create Apache virtual host configuration** at `/etc/apache2/sites-available/myapp.conf`:
   ```apache
   <VirtualHost *:80>
       ServerName example.com
       
       ProxyPreserveHost On
       ProxyPass / http://127.0.0.1:5000/
       ProxyPassReverse / http://127.0.0.1:5000/
       
       RequestHeader set X-Forwarded-Proto "http"
   </VirtualHost>
   ```

8. **Enable the site and verify configuration:**
   ```bash
   sudo a2ensite myapp.conf
   sudo apache2ctl configtest
   sudo systemctl restart apache2
   ```

## Process Signatures

- **Process name / executable:** `dotnet` (the .NET runtime executable)
- **Command-line patterns:** `/usr/bin/dotnet /var/www/<appname>/<appname>.dll` or similar paths pointing to a .dll file in the application directory
- **Parent process:** `systemd` (PID 1 or a systemd user service manager)
- **Typical user:** `www-data` (on Ubuntu/Debian) or `apache` (on CentOS/RHEL)
- **Working directory:** `/var/www/<appname>` or `/opt/<appname>` (the directory containing the published .dll and supporting files)

## File System Paths

### Linux

- **Install root:** `/var/www/<appname>` (typical; may vary) or `/opt/<appname>`
- **Binaries:** Kestrel is embedded in the published .NET assembly; the entry point is `<appname>.dll` in the application directory
- **Application/deploy dir:** `/var/www/<appname>` (contains the published .dll, appsettings.json, and supporting files)
- **PID file:** N/A (systemd manages the process lifecycle; PID is tracked by systemd)
- **Apache configuration:** `/etc/apache2/sites-available/<appname>.conf` (Debian/Ubuntu) or `/etc/httpd/conf.d/<appname>.conf` (CentOS/RHEL)
- **systemd unit file:** `/etc/systemd/system/kestrel-<appname>.service`

## Environment Variables

| Variable | Purpose | Typical value |
|---|---|---|
| ASPNETCORE_ENVIRONMENT | Sets the application environment (Development, Staging, Production) | `Production` |
| ASPNETCORE_URLS | Specifies the URLs and ports Kestrel listens on | `http://127.0.0.1:5000` |
| DOTNET_ROOT | Path to the .NET runtime installation | `/usr/share/dotnet` |
| DOTNET_SKIP_FIRST_RUN_EXPERIENCE | Skips telemetry and first-run setup | `true` |
| DOTNET_CLI_HOME | Home directory for .NET CLI | `/var/www/<appname>` or N/A |

## Configuration Files

- **`appsettings.json`** — Application configuration file (connection strings, logging, feature flags); located in the application directory (e.g., `/var/www/myapp/appsettings.json`)
- **`appsettings.Production.json`** — Environment-specific overrides for production; same directory
- **Apache virtual host configuration** — Path depends on distribution:
  - Debian/Ubuntu: `/etc/apache2/sites-available/<appname>.conf`
  - CentOS/RHEL: `/etc/httpd/conf.d/<appname>.conf`
- **systemd unit file** — `/etc/systemd/system/kestrel-<appname>.service`

## Log Locations

- **systemd journal:** Captured by systemd if `SyslogIdentifier=dotnet-<appname>` is set in the unit file; view with `journalctl -u kestrel-<appname>.service -f`
- **Application logs:** Written to files specified in `appsettings.json` (typically in `/var/www/<appname>/logs/` if configured)
- **Apache access/error logs:**
  - Debian/Ubuntu: `/var/log/apache2/access.log` and `/var/log/apache2/error.log`
  - CentOS/RHEL: `/var/log/httpd/access_log` and `/var/log/httpd/error_log`

## Service / Init Integration

The ASP.NET Core Kestrel process is managed by **systemd**. The unit file is located at:

```
/etc/systemd/system/kestrel-<appname>.service
```

Example unit name: `kestrel-myapp.service`

**Key systemd commands:**
- `sudo systemctl start kestrel-<appname>.service` — Start the service
- `sudo systemctl stop kestrel-<appname>.service` — Stop the service
- `sudo systemctl restart kestrel-<appname>.service` — Restart the service
- `sudo systemctl status kestrel-<appname>.service` — View current status
- `sudo systemctl enable kestrel-<appname>.service` — Enable automatic startup on boot
- `sudo systemctl disable kestrel-<appname>.service` — Disable automatic startup
- `journalctl -u kestrel-<appname>.service -f` — Stream logs in real-time

The `Type=notify` or `Type=simple` setting in the `[Service]` section determines how systemd detects successful startup. The `Restart=always` directive ensures the process automatically restarts on crash or unexpected termination.

## Detection Heuristics

**Highest-confidence signals (in order):**

1. **Process executable and command-line:** Running process named `dotnet` with a command-line argument pointing to a `.dll` file in a known application directory (e.g., `/var/www/*/` or `/opt/*/`)
   - Example: `/usr/bin/dotnet /var/www/myapp/MyApp.dll`

2. **Environment variable:** Presence of `ASPNETCORE_ENVIRONMENT` or `ASPNETCORE_URLS` in the process environment; `ASPNETCORE_ENVIRONMENT=Production` is typical

3. **systemd unit file:** Existence of `/etc/systemd/system/kestrel-*.service` with `ExecStart` pointing to a `dotnet` binary and a `.dll` file

4. **Listening port:** Process listening on `127.0.0.1:5000` (or other high-numbered port) with Apache process (`httpd` or `apache2`) listening on `0.0.0.0:80` or `0.0.0.0:443` on the same host

5. **Working directory:** Process running from a directory under `/var/www/`, `/opt/`, or similar application roots, containing a `.dll` file matching the process name

6. **Shared libraries:** Link maps show `libcoreclr.so`, `System.*.dll`, or other .NET Core runtime libraries

**Strongest detection combination:** Match `dotnet` process + presence of `ASPNETCORE_ENVIRONMENT` env var + corresponding `/etc/systemd/system/kestrel-*.service` unit file

## Version / Variant Differences

- **ASP.NET Core versions:** 1.0–10.0; all follow the same systemd + Apache pattern. Version differences are in the published assembly and dependencies but do not affect process detection.
- **Linux distributions:** Ubuntu/Debian use `www-data` as the default service user and Apache is `apache2`; CentOS/RHEL use `apache` user and Apache is `httpd`. These are cosmetic differences in user/package names; detection is identical.
- **Kestrel port:** Default is 5000, but may be set to 5001 or another port via `appsettings.json` or `ASPNETCORE_URLS` env var. Detection should check the `ASPNETCORE_URLS` variable or examine process network bindings.
- **Reverse proxy variant:** Some setups use Nginx instead of Apache; this skill covers Apache only. Nginx deployments are a separate variant.
- **systemd Type:** `Type=simple` or `Type=notify` are both common; detection is unaffected.

## Sources

- [Microsoft Docs: Host ASP.NET Core on Linux with Apache](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/linux-apache)
- [Code Maze: Deploy ASP.NET Core Applications on Linux With Apache](https://code-maze.com/aspnetcore-deploy-applications-on-linux-with-apache/)
- [0x191 Unauthorized: Host ASP.NET Core on Linux with Apache and Kestrel](https://0x191unauthorized.blogspot.com/2021/04/host-aspnet-core-on-linux-with-apache.html)
- [Syncfusion Blogs: Hosting Multiple ASP.NET Core Apps in Ubuntu Linux Server Using Apache](https://www.syncfusion.com/blogs/post/hosting-multiple-aspnet-core-apps-in-ubuntu-linux-server-using-apache)
- [Medium: How to host an ASP.NET Core 3.1 application on Linux Ubuntu 20.04 with Apache as reverse proxy](https://ollie10.medium.com/how-to-host-an-asp-net-core-3-1-application-on-linux-ubuntu-20-04-with-apache-as-reverse-proxy-34bf2fb18502)
