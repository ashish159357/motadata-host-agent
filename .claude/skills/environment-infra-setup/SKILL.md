---
name: environment-infra-setup
description: Install and configure runtime and reverse proxy for a no-DB deployment skill. Called by the environment-setup agent for each matched deployment skill. Enforces strict provisioning order, checks before installing, health-checks after each component, and escalates to environment-setup-troubleshooter on repeated failure.
---

# Skill: environment-infra-setup

## ⚠️ Invocation Guard

This skill is called **only by the `environment-setup` agent**. It is never invoked directly by the user.

If this skill is triggered without a full parameter set, stop immediately and respond:

> "This skill is internal and must be invoked by the environment-setup agent, not directly. Run `/start-environment-setup` to start the full pipeline."

Do not ask the user for any inputs. Do not proceed without a complete parameter set from the calling agent.

---

You are given a parsed deployment skill. Your job is to install and configure every infrastructure component on the target server so it matches the architecture defined in the deployment skill.

## Input (from calling agent)
- `runtime` — language + version (e.g., "Java 17", "Go 1.22")
- `language` — lowercase language name (e.g., `java`, `go`, `python`)
- `runtime_install_commands` — ordered list of commands to install the runtime if not already present
- `components` — list of all components, each with: `name`, `version`, `port`, `bind_address`
- `system_packages` — list of packages to install via the OS package manager
- `package_manager` — OS package manager (`apt`, `yum`, `dnf`, `brew`)
- `reverse_proxy_type` — reverse proxy name (e.g., `nginx`, `caddy`, `apache`) — omit if none
- `reverse_proxy_config` — full reverse proxy server block string
- `reverse_proxy_config_path` — directory to write the site config (e.g., `/etc/nginx/sites-available`)
- `reverse_proxy_enable_path` — symlink target directory (e.g., `/etc/nginx/sites-enabled`)
- `app_name` — used for reverse proxy site config filename
- `app_port` — port the app listens on
- `verification_commands` — ordered list of commands to verify the full setup

**No database inputs.** These are no-DB deployments — never install, configure, or connect to any database.

---

## Execution Steps

### Step 1 — Detect Package Manager & Update Index

Detect the available package manager on the current OS:
```bash
if command -v apt-get &>/dev/null; then PM="apt-get"; PM_UPDATE="apt-get update -qq"; PM_INSTALL="apt-get install -y"
elif command -v yum &>/dev/null;     then PM="yum";     PM_UPDATE="yum check-update -q"; PM_INSTALL="yum install -y"
elif command -v dnf &>/dev/null;     then PM="dnf";     PM_UPDATE="dnf check-update -q"; PM_INSTALL="dnf install -y"
elif command -v brew &>/dev/null;    then PM="brew";    PM_UPDATE="brew update";          PM_INSTALL="brew install"
else echo "ERROR: No supported package manager found"; exit 1
fi

sudo $PM_UPDATE
```

If update fails: report and stop — do not continue without an updated package index.

---

### Step 2 — Port Conflict Pre-Check

Before installing anything, verify all ports in `components` are free:
```bash
for port in {each port from components}; do
  ss -tlnp | grep ":$port " && echo "CONFLICT: $port" || echo "FREE: $port"
done
```

If any port is occupied by an incompatible process:
- Identify owner: `lsof -i :{port}`
- Report to log — do NOT kill without user confirmation
- Stop provisioning this skill if conflict cannot be resolved

---

### Step 3 — Install System Packages

For each package in `system_packages`:
1. Check if already installed at the correct version:
   ```bash
   # apt: dpkg -s {package} | grep "Status: install ok installed"
   # yum/dnf: rpm -q {package}
   # brew: brew list {package}
   ```
2. If missing or wrong version: `sudo $PM_INSTALL {package}`
3. Verify install succeeded before moving to the next package

---

### Step 4 — Install Runtime

1. Check if runtime binary is already at the required version:
   ```bash
   {runtime_binary} --version 2>/dev/null
   ```
2. If missing or wrong version: run `runtime_install_commands` in order:
   ```bash
   {runtime_install_command_1}
   {runtime_install_command_2}
   ...
   ```
3. Verify version after install:
   ```bash
   {runtime_binary} --version
   ```
4. If multiple versions exist: set the required version as the system default using the appropriate tool (update-alternatives, update-java-alternatives, etc.)

---

### Step 5 — Configure Supporting Services

For each service in `components` that is not the runtime or DB:
1. Check if already installed and running at the correct version
2. If missing: install via `system_packages` or component-specific install command
3. Start and enable the service:
   ```bash
   sudo systemctl start {service_name} && sudo systemctl enable {service_name}
   ```
4. Health-check:
   ```bash
   systemctl is-active {service_name}
   ```
   If not active after start: invoke **environment-setup-troubleshooter**.

---

### Step 6 — Configure Reverse Proxy

Skip this step if `reverse_proxy_type` is not defined.

1. Check if reverse proxy is installed and running — install and start if not
2. Write site config to `{reverse_proxy_config_path}/{app_name}` (overwrite only if content differs)
3. Enable site:
   ```bash
   sudo ln -sf {reverse_proxy_config_path}/{app_name} {reverse_proxy_enable_path}/{app_name}
   ```
4. Remove default conflicting config if it occupies the same port
5. Test config:
   ```bash
   sudo {reverse_proxy_type} -t    # nginx / apache
   # caddy: caddy validate --config {config_path}
   ```
   If test fails: invoke **environment-setup-troubleshooter** with full test output
6. Reload:
   ```bash
   sudo systemctl reload {reverse_proxy_type}
   ```

---

### Step 7 — Run Verification Commands

Run each command in `verification_commands` in order:
- Print command + full output
- Mark PASS or FAIL
- Collect all failures — do not stop on individual failure
- After all checks: if any FAIL → invoke **environment-setup-troubleshooter** with the full failure list

---

## Idempotency Rules
- Never overwrite a config without first checking if content is already identical
- Never reinstall a package already at the correct version
- Re-running on an already-provisioned server must produce no errors and no changes

## Escalation
After 2 consecutive failures on any single step, stop and call **environment-setup-troubleshooter** with:
- Step name
- Command attempted
- Full stdout + stderr
- Current service statuses (`systemctl is-active` for each component)
