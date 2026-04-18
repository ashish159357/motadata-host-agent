---
name: environment-setup-troubleshooter
description: Diagnose and recover from failures during environment provisioning or app scaffolding. Called by environment-setup, environment-infra-setup, or environment-app-setup when a step fails twice, a port conflict exists, a service won't start, or verification fails. Collects system state, identifies root cause, attempts safe fixes, and escalates to the user if auto-fix is not safe.
---

# Skill: environment-setup-troubleshooter

## ⚠️ Invocation Guard

This skill is called **only by `environment-setup`, `environment-infra-setup`, or `environment-app-setup`**. It is never invoked directly by the user.

If this skill is triggered without a full parameter set (failed step, command, error output), stop immediately and respond:

> "This skill is internal and must be invoked by an environment agent, not directly. Run `/start-environment-setup` to start the full pipeline."

Do not ask the user for any inputs. Do not proceed without a complete parameter set from the calling agent.

---

You are invoked when something is stuck or broken during environment setup. Diagnose the root cause, attempt a safe fix, and either resolve it or give the user one clear action to take.

## Input (from calling skill or agent)
- `failed_step` — which step failed (e.g., "environment-infra-setup Step 5 Database")
- `command` — the exact command that failed
- `stdout` — command stdout
- `stderr` — command stderr
- `retry_count` — how many times this step was retried before escalating
- `package_manager` — detected package manager (`apt`, `yum`, `dnf`, `brew`)
- `app_port` — app port
- `app_name` — app name
- `app_output_path` — resolved app output path
- `reverse_proxy_type` — reverse proxy type (e.g., `nginx`, `caddy`, `apache`) — omit if none
- `network_ip` — server LAN IP

## Return Contract

After completing diagnosis and any auto-fix, return:
```
OUTCOME: RESOLVED | STILL_FAILING
FIXED_STEP: {step name that was fixed, if resolved}
ACTION_TAKEN: {what was done}
NEXT_STEP: {exact command or instruction for the user, if still failing}
```

The calling skill uses `OUTCOME` to decide whether to retry the failed step.

---

## Diagnosis Runbook

### 1. Collect System State

```bash
# Disk space
df -h / | tail -1

# Memory
free -m | grep Mem

# All listening ports
ss -tlnp

# Running services
systemctl list-units --type=service --state=running 2>/dev/null | grep -E "{reverse_proxy_type}|{app_name}"

# Recent system errors
journalctl -n 50 --no-pager -p err 2>/dev/null | tail -20

# Package manager lock
case "{package_manager}" in
  apt)     lsof /var/lib/dpkg/lock-frontend 2>/dev/null && echo "PKG LOCKED" || echo "PKG FREE" ;;
  yum|dnf) lsof /var/run/yum.pid 2>/dev/null && echo "PKG LOCKED" || echo "PKG FREE" ;;
  brew)    echo "brew: no lock file" ;;
esac
```

---

### 2. Diagnose by Failure Category

#### A. Package Install Failure

Refresh index and fix broken state:
```bash
case "{package_manager}" in
  apt)  sudo apt-get update && sudo apt-get install -f -y ;;
  yum)  sudo yum clean all && sudo yum makecache ;;
  dnf)  sudo dnf clean all && sudo dnf makecache ;;
  brew) brew update && brew doctor ;;
esac
```

Check for:
- Package manager lock: remove lock file only if no package manager process is running
- Missing repository: re-add the required repo using the method from the deployment skill, then retry

---

#### B. Runtime Installation Failure

```bash
# Check if binary exists anywhere on the system
which {runtime_binary} 2>/dev/null
find /usr /opt /home -name "{runtime_binary}" 2>/dev/null | head -5

# Check current PATH
echo $PATH
```

Common fixes:
- Installed but not in PATH → add install directory to PATH and to shell profile
- Wrong version → use version manager or `update-alternatives` to switch
- Install failed mid-way → re-run `runtime_install_commands` from scratch

---

#### C. Port Already in Use

```bash
lsof -i :{port}
ss -tlnp | grep ":{port} "
```

**Never kill a process without telling the user.** Instead:
- Report: `Port {port} is held by PID {pid} ({process_name})`
- Suggest: `sudo kill {pid}` or reconfigure app to an alternate port
- Exception: if the process is a previous run of this same deployment — safe to kill after reporting

---

#### D. Reverse Proxy Config Test Fails

```bash
# nginx:
sudo nginx -t 2>&1
# apache:
sudo apachectl configtest 2>&1
# caddy:
caddy validate --config {config_path} 2>&1
```

Read the failing config file:
```bash
cat {reverse_proxy_config_path}/{app_name}
```

Common causes and fixes:
- Syntax error (missing semicolon, bracket) → fix and re-test
- Wrong proxy target port → correct to `{app_port}`
- Duplicate listener → remove conflicting default config

After fix: re-test config, then reload if passing.

---

#### E. App Won't Start / Health Check Fails

```bash
# Check process manager logs
journalctl -u {app_name} -n 50 --no-pager 2>/dev/null
pm2 logs {app_name} --lines 50 2>/dev/null

# Check if process is running at all
ps aux | grep {app_name} | grep -v grep
ss -tlnp | grep ":{app_port}"
```

Common causes:
- Port already in use → see Section D
- Missing runtime config file → verify the runtime config file exists at the expected path for this language/runtime (e.g. `appsettings.json` for .NET, `start.sh` env vars for Go/Rust)
- Missing dependency → re-run `build_commands`
- Wrong working directory → app must start from `{app_output_path}`
- App binding to `127.0.0.1` → fix `start_command` or app config to bind `0.0.0.0`

---

#### F. App Running but Not Reachable via Reverse Proxy

```bash
# Verify app is listening
ss -tlnp | grep ":{app_port}"

# Test app directly (bypass proxy)
curl -v http://localhost:{app_port}/health 2>&1

# Test through proxy
curl -v http://{network_ip}/health 2>&1

# Check proxy site is enabled
ls -la {reverse_proxy_enable_path}/
```

Fixes:
- Proxy `proxy_pass` port doesn't match `{app_port}` → fix config and reload
- App binding to `127.0.0.1` not `0.0.0.0` → fix start command or app config
- Site config not enabled → symlink to enable path and reload

---

### 3. Auto-Fix Rules

**Safe to auto-fix (do it, then report):**
- Reload or restart a service after a config change
- Fix reverse proxy config syntax errors
- Remove stale package manager lock if no package manager process is running
- Re-run `build_commands` if dependencies are missing
- Add runtime binary to PATH if installed but not found

**NOT safe to auto-fix (report to user, ask for confirmation):**
- Killing a process not started by this provisioning session
- Changing a port defined in the deployment skill
- Modifying SSH config or firewall rules
- Changing file ownership outside `{app_output_path}`

---

### 4. Recovery Report Format

```
FAILURE: {failed_step}
   Command:  {command}
   Error:    {stderr summary}

DIAGNOSIS:
   Root cause: {one-line explanation}
   System state: {disk / memory / port conflicts / service statuses}

ACTION TAKEN:
   {exactly what was done — or "none, requires user action"}
   Result: RESOLVED / STILL FAILING

NEXT STEP (if still failing):
   Run: {exact single command}
   Or:  {one clear human instruction}

OUTCOME: RESOLVED | STILL_FAILING
```

---

### 5. Escalation

If after one auto-fix attempt the problem persists:
1. Print the full recovery report
2. Return `OUTCOME: STILL_FAILING` to the calling skill
3. The calling skill stops provisioning for this deployment and moves to the next skill
4. Offer to resume from the failed step once the user confirms it's resolved
