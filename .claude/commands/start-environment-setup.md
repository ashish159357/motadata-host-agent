---
description: Phase 2 — Analyse the environment, provision apps, then offer to start all deployed apps
---

You orchestrate Phase 2 of the pipeline in three stages: **Analysis** → **Setup** → **Start**. Always run in order; never skip ahead without the user confirming each gate.

---

## Phase 1 — Analysis (you do this inline)

### Step 1 — Detect OS

```bash
uname -s && uname -m
cat /etc/os-release 2>/dev/null
sw_vers 2>/dev/null
hostname
```

Extract and record: `os_type`, `distro`, `distro_version`, `arch`, `hostname`.

### Step 2 — Collect and Classify Deployment Skills

```bash
find .claude/skills -name "SKILL.md" | sort
```

For each `SKILL.md`, read its frontmatter `name` and `description`. Classify as:
- **Compatible** — description mentions the detected `os_type` / `distro`
- **Incompatible** — targets a different OS
- **Internal** — helper skills (`environment-*`, `detect-language`) — always skip

### Step 3 — Check Already-Deployed Apps

For each **Compatible** skill, resolve its app path and check if it is already running.

```bash
CLAUDE_DIR=$(find . -maxdepth 3 -type d -name ".claude" | head -1)
PROJECT_DIR=$(dirname "$CLAUDE_DIR")
APP_PATH="$PROJECT_DIR/apps/{language}/{deployment_type}"
```

**For HTTP apps (have a port):**
```bash
[ -d "$APP_PATH" ] && DIR_EXISTS=true
ss -tlnp | grep -q ":{app_port} " && PORT_UP=true
curl -sf --max-time 3 http://localhost:{app_port}/health && HEALTH_OK=true
```

**For background services (no port):**
```bash
[ -d "$APP_PATH" ] && DIR_EXISTS=true
systemctl --user is-active {service_unit_name} 2>/dev/null | grep -q "^active$" && SERVICE_ACTIVE=true
```

Decision:
- `DIR_EXISTS + PORT_UP + HEALTH_OK` (or `DIR_EXISTS + SERVICE_ACTIVE`) → **Already Deployed**
- Everything else → **To Provision**

### Step 4 — Print Analysis Report

```
## Environment Analysis

### System
| Field        | Value                         |
|--------------|-------------------------------|
| OS           | {os_type}                     |
| Distro       | {distro} {distro_version}     |
| Architecture | {arch}                        |
| Hostname     | {hostname}                    |

### Deployment Skills

#### 🚀 To Provision (compatible, not yet running)
- {skill-name} — {one-line description}

#### ✅ Already Deployed (healthy — will be skipped)
- {skill-name} — {health_url} | health OK  (or: service active — no HTTP endpoint)

#### ⚠️ Incompatible (wrong OS — will be skipped)
- {skill-name} — targets {other OS}
```

If nothing is "To Provision": skip Phase 2, go directly to Phase 3.
If no compatible skills exist at all: tell the user to run `/start-research <language>` first and stop.

### Step 5 — Gate 1

> "Reply `go` to provision the **To Provision** skills above, or `skip` to go straight to starting apps, or `cancel` to abort."

**Stop here and wait.** Do not proceed without the user's reply.

---

## Phase 2 — Setup (only after user says `go`)

Spawn the `environment-setup` agent with `subagent_type: "environment-setup"` and this prompt:

> "Run the full environment setup pipeline as defined in your agent instructions. Process only these skills: `{to-provision skill names}`. For each: invoke `environment-infra-setup` then `environment-app-setup`. Write the execution log and `docs/environment/current-environment.md` as specified. Skip any skill not in this list."

Wait for the agent to complete. Relay its summary table (SUCCESS / FAILED / SKIPPED). On failures, point the user to the execution log.

Then proceed directly to Phase 3.

---

## Phase 3 — Start All Apps

Read `docs/environment/current-environment.md`. Extract every `systemd Unit` field from all deployed applications — these are the exact unit names to manage.

Check current status of each unit:

```bash
# For each unit extracted from current-environment.md:
systemctl --user is-active {unit-name} 2>/dev/null
```

Build two lists:
- **Running** — `is-active` returns `active`
- **Stopped** — anything else (`inactive`, `failed`, `unknown`)

Also extract `Health Endpoint` for each app from `current-environment.md` — this is the running URL to display.

Print the status table:

```
## App Status

| App | systemd Unit | URL | Status |
|-----|-------------|-----|--------|
| {skill-name} | {unit} | {health_url or "no HTTP endpoint"} | 🟢 Running / 🔴 Stopped |
```

For running HTTP apps, also confirm the URL is reachable:
```bash
curl -sf --max-time 3 {health_url}
```
Show the health response inline next to the URL if reachable.

### Gate 2

If any apps are **Stopped**:

> "X app(s) are stopped. Reply `start` to start all stopped apps now, or `skip` to leave them as-is."

If all apps are already **Running**:

> "All apps are running. No action needed."
> Prompt: "Run `/start-code` to move to Phase 3."
> Stop.

**Wait for user reply.**

### On `start`

For each stopped unit, run:
```bash
systemctl --user start {unit-name}
sleep 2
systemctl --user is-active {unit-name}
```

For HTTP apps, also run:
```bash
curl -sf --max-time 5 http://localhost:{port}/health
```

Print final confirmation table showing each unit's new status. Then prompt:

> "All apps started. Run `/start-code` to move to Phase 3."

---

## Rules

- **Never skip Phase 1.** Always run analysis first.
- **Never skip Gate 1.** Never provision without the user saying `go` or `skip`.
- **Always run Phase 3** — whether Phase 2 ran or not.
- **Never pass already-deployed skills to the setup agent.**
- If the user says `cancel` at any gate, confirm and stop.
