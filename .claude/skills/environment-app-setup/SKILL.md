---
name: environment-app-setup
description: Scaffolds and starts a minimal no-DB sample application to demonstrate deployment architecture for process detection. Exposes /health, binds to the configured port, and is detectable via /proc. Called by environment-setup after environment-infra-setup completes.
---

# Skill: environment-app-setup

## ⚠️ Invocation Guard

This skill is called **only by the `environment-setup` agent**. It is never invoked directly by the user.

If this skill is triggered without a full parameter set (runtime, language, deployment_type, app_port, etc.), stop immediately and respond:

> "This skill is internal and must be invoked by the environment-setup agent, not directly. Run `/start-environment-setup` to start the full pipeline."

Do not ask the user for any inputs. Do not proceed without a complete parameter set from the calling agent.

---

## Purpose

Build and run a **minimal no-DB sample application** — not a production app. The sole purpose is to demonstrate the deployment architecture so the `motadata-host-agent` binary can detect it via `/proc`.

The app only needs to:
1. Start and stay running on the configured port
2. Expose a `/health` endpoint that returns a basic JSON response
3. Be detectable by process name, environment variables, and port via `/proc`

**No database. No auth. No business logic. No migrations.** Do not add a DB dependency even if one is provisioned — it adds failure surface with zero detection benefit. A single-file HTTP server that responds to `/health` is the target complexity.

## Input (from calling agent)
- `runtime` — language + version (from deployment skill)
- `language` — lowercase language name (e.g., `java`, `go`, `dotnet`, `python`)
- `deployment_type` — deployment type name matching the skill directory (e.g., `jboss`, `tomcat`)
- `app_name` — name of the application
- `app_files` — list of files to create, each with `path` (relative) and `content` — omit if deployment skill handles generation differently
- `app_port` — port the app must bind to
- `env_vars` — key-value map of runtime env vars required by the app (no DB vars)
- `build_commands` — ordered list of commands to install dependencies and build
- `start_command` — command to start the app (must bind to `0.0.0.0:{app_port}`)
- `stop_command` — command to stop the app (e.g., `pkill -f {app_name}`, `pm2 stop {app_name}`)
- `startup_wait_seconds` — seconds to wait after start before health check (default: `10`)
- `process_manager` — process manager config: `tool` (pm2/systemd/supervisor), `service_name`, `auto_restart`
- `health_check_command` — command to verify app is running (e.g., `curl -sf http://localhost:{app_port}/health`)
- `health_check_expected` — expected output or exit code that indicates success

## App Requirements
- **Minimal**: single-file or smallest possible structure for the runtime — no unnecessary complexity
- **Network bind**: must listen on `0.0.0.0:{app_port}` (not 127.0.0.1) so it is detectable from the network
- **Health endpoint**: expose `/health` returning `{"status":"ok","runtime":"{runtime}","deployment":"{deployment_type}"}`
- **No database** — do not add any DB dependency, driver, or connection logic under any circumstance
- **Process name**: must be identifiable by process name or working directory for detection purposes

---

## Execution Steps

### Step 1 — Resolve and Prepare Output Directory

```bash
# Locate project root (parent of .claude)
CLAUDE_DIR=$(find . -maxdepth 3 -type d -name ".claude" | head -1)
PROJECT_DIR=$(dirname "$CLAUDE_DIR")

# Build path: {project_root}/apps/{language}/{deployment_type}
APP_OUTPUT_PATH="$PROJECT_DIR/apps/{language}/{deployment_type}"

# Create directory (intermediate dirs created automatically)
mkdir -p "$APP_OUTPUT_PATH"
```

Use `$APP_OUTPUT_PATH` as the working directory for all subsequent steps.

**Idempotency**: if `$APP_OUTPUT_PATH` already exists and the app is running on `app_port`, skip Steps 2–5 and go directly to Step 6 (health check only).

---

### Step 2 — Configure Runtime Environment

Write configuration **before** scaffolding or building — builds may depend on these values.

These apps have no database. The only config needed is `app_port` and any runtime-specific env vars (e.g. `ASPNETCORE_ENVIRONMENT`, `ASPNETCORE_URLS`).

**Do not use a hardcoded format.** Use your knowledge of `{language}` and `{runtime}` to determine:

1. **Whether a config file is needed at all** — some runtimes (Go, Rust) need no config file; port and env vars are passed via the process environment in `start.sh`.

2. **What format the runtime natively reads** — write only what the runtime will actually load (e.g. `appsettings.json` for .NET, not `.env`).

3. **Where that file must live** — root, `src/main/resources/`, `config/`, etc. based on the runtime's conventions.

**Constraints:**
- No DB keys, no connection strings — this app has no database
- Write only one config artifact — the one the runtime will actually load at startup
- If the runtime needs no config file, set env vars in `start.sh` only
- If the file already exists with identical content: skip the write

---

### Step 3 — Scaffold Application Files

If `app_files` is provided: write each file to `$APP_OUTPUT_PATH`:
```
for each file in app_files:
  - create parent subdirectories if needed
  - write file content
  - skip if file already exists with identical content
```

If `app_files` is not provided: the deployment skill itself defines the generation approach — follow the skill's instructions for creating the app structure (e.g., running a project generator command, unpacking a template, etc.).

---

### Step 4 — Install Dependencies & Build

Run `build_commands` in order from `$APP_OUTPUT_PATH`:
```bash
cd "$APP_OUTPUT_PATH"
{build_command_1}
{build_command_2}
...
```

On any command failure: invoke **environment-setup-troubleshooter** with the failed command and full stdout + stderr. Do not continue to the next build command if one fails.

---

### Step 5 — Start the Application

**5a** — If app is already running on `app_port`: stop it first using `stop_command` (only if it belongs to this deployment — verify by checking the process working directory or name).

**5b** — Start the app:
```bash
cd "$APP_OUTPUT_PATH"
{start_command}
```

**5c** — If `process_manager` is defined: register the app with the process manager for auto-restart on reboot:
```bash
# pm2 example:
pm2 start {start_command} --name {process_manager.service_name}
pm2 save

# systemd example: write a unit file, then:
sudo systemctl daemon-reload
sudo systemctl enable {process_manager.service_name}
sudo systemctl start {process_manager.service_name}
```

---

### Step 6 — Post-Start Health Check

Wait for app to become ready:
```bash
sleep {startup_wait_seconds}   # use value from input, default 10
```

Run health check:
```bash
{health_check_command}
```

Compare result against `health_check_expected`:
- Match → log SUCCESS
- No match → invoke **environment-setup-troubleshooter**, then retry health check once after resolution
- Still failing → log FAILED, report to agent

---

## Idempotency Rules
- If app directory already exists and app is running healthy: skip Steps 2–5, run Step 6 only
- If app directory exists but app is stopped: skip Steps 2–3 (files exist), re-run Steps 4–6 only
- Never stop a running app without verifying it belongs to this deployment

## Escalation
On any step failure after 1 retry: invoke **environment-setup-troubleshooter** with:
- Step name
- Command attempted
- Full stdout + stderr
