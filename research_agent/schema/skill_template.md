---
name: <skill-name>
description: <one-sentence description optimized for detection. Mention the tech, OS, init system/variant, and 2-3 concrete signals (process names, env vars, config files, paths) that unambiguously identify this deployment.>
---

# <Tech> on <OS> — <Variant>

## Overview
<2-4 sentences: what this deployment pattern is, when teams choose it, how common it is in production.>

## Deployment Process
<Ordered steps an operator runs to deploy an app this way. Include the command invocations.>

## Process Signatures
- **Process name / executable:** <e.g. `java`, `node`, `dotnet`>
- **Command-line patterns:** <representative argv patterns a detector can match on>
- **Parent process:** <e.g. `systemd`, `services.exe`, a shell>
- **Typical user:** <e.g. `tomcat`, `www-data`, `SYSTEM`>
- **Working directory:** <where the process is usually launched from>

## File System Paths

### Linux
- **Install root:** <e.g. `/opt/tomcat`, `/usr/share/tomcat9`>
- **Binaries:** <paths>
- **Application/deploy dir:** <paths>
- **PID file:** <path if any>

### Windows
- **Install root:** <e.g. `C:\Program Files\...`>
- **Binaries:** <paths>
- **Application/deploy dir:** <paths>

(Omit a subsection if the variant only applies to one OS.)

## Environment Variables
| Variable | Purpose | Typical value |
|---|---|---|
| <NAME> | <what it does> | <example> |

## Configuration Files
- **<filename>** — <purpose, absolute path>
- ...

## Log Locations
- <absolute paths to log files, rotation patterns if relevant>

## Service / Init Integration
<How the process is supervised: systemd unit file name and path, Windows service name, init.d script, launchd plist, etc. Include the exact unit/service identifier.>

## Detection Heuristics
<The highest-confidence signals a detection tool can look for. Prefer combinations that are unambiguous. Example: "presence of CATALINA_HOME env var on a java process whose parent is systemd and whose unit file is /etc/systemd/system/tomcat.service".>

## Version / Variant Differences
<Anything that varies across versions or sub-variants that a detector must handle.>

## Sources
- <URL 1>
- <URL 2>
