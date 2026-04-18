# .claude/ Architecture

Complete guide to how commands, agents, and skills work together in the Claude Code harness.

---

## Overview

The `.claude/` folder orchestrates a 4-phase pipeline for language detection development:

```
Phase 1: Research        Phase 2: Provision        Phase 3: Code          Phase 4: Test
┌──────────────────┐    ┌──────────────────────┐  ┌──────────────────┐   ┌──────────────────┐
│ /start-research  │───▶│ /start-environment-  │─▶│ /start-code      │──▶│ /start-test      │
│                  │    │  setup               │  │                  │   │                  │
│ Research host/VM │    │ Provision host and   │  │ Close detection  │   │ Run agent binary │
│ deployment       │    │ deploy a real        │  │ gaps in          │   │ against deployed │
│ patterns per     │    │ application using    │  │ internal/system, │   │ app; verify it   │
│ language.        │    │ Phase 1 skills.      │  │ rebuild binary.  │   │ detects language.│
│                  │    │                      │  │                  │   │                  │
│ Output:          │    │ Output:              │  │ Output:          │   │ Output:          │
│ .claude/skills/  │    │ docs/environment/    │  │ bin/motadata-    │   │ PASS/FAIL report │
│ *-SKILL.md       │    │ current-             │  │ host-agent       │   │ with reasons for │
│                  │    │ environment.md       │  │ (ELF binary)     │   │ any miss.        │
└──────────────────┘    └──────────────────────┘  └──────────────────┘   └──────────────────┘
```

---

## Three-Layer Model

### Layer 1: Commands (Entrypoints)

**Location:** `.claude/commands/`

Commands are user-facing slash commands that delegate work to agents.

| Command | File | Invokes Agent | Purpose |
|---------|------|---------------|---------|
| `/start-code <lang>` | `start-code.md` | `add-detect-language-code` | Add language detection signals |
| `/start-environment-setup` | `start-environment-setup.md` | `environment-setup` | Provision test environment |
| `/start-test` | `start-test.md` | `test-language-detection` | Run binary against deployed app |
| `/start-research <lang>` | `start-research.md` | `deployment-researcher` | Research deployment patterns |
| `/list-languages` | `list-languages.md` | `language-lister` | Report coverage status |

**Structure of a command file:**
```markdown
---
description: One-line summary of what the command does
---

## Usage
/command [args]

## Execution
Follow the full instructions in: .claude/agents/agent-name.md
```

**Key point:** Commands are thin wrappers that validate arguments early, then invoke the agent.

---

### Layer 2: Agents (Execution Logic)

**Location:** `.claude/agents/`

Agents contain the detailed procedural logic. They read files, make decisions, spawn other agents, and coordinate the pipeline.

#### Core Agents

| Agent | Spawns | Responsibilities |
|-------|--------|------------------|
| `add-detect-language-code.md` | None | Gap analysis, signal addition, test writing, build verification, registry updates |
| `environment-setup.md` | `environment-infra-setup`, `environment-app-setup` | OS detection, skill classification, app provisioning orchestration |
| `test-language-detection.md` | None | Binary execution, output parsing, PASS/FAIL validation |
| `deployment-researcher.md` | None | Host/VM research, skill documentation generation |
| `language-lister.md` | None | Registry reading, coverage classification, report generation |

**Agent responsibilities example (add-detect-language-code):**
1. Parse `$ARGUMENTS` and validate language
2. Read `LANGUAGE_DETECTION_REGISTRY.md` (current coverage)
3. Read `internal/system/language.go` (live source)
4. Cross-reference against Signal Catalogue in `skills/detect-language/SKILL.md`
5. Identify gaps and add missing signals
6. Write test cases to `language_test.go`
7. Run `gofmt`, `go test`, `go build` — stop on first failure
8. Update registries and generate summary report

---

### Layer 3: Skills (Reusable Patterns)

**Location:** `.claude/skills/*/SKILL.md`

Skills encode reusable knowledge about:
- **Detection patterns:** signals for identifying a language across 5 stages (environ → cmdline → files → maps → exepath)
- **Deployment patterns:** how to provision and supervise an application on a specific OS/runtime combo

#### Skill Categories

##### Detection Skills (Used by Phase 3)
- `detect-language/SKILL.md` — Signal catalogue for one language across all 5 stages

**Example entry:**
```yaml
### golang
| Stage | Signals |
|-------|---------|
| Environ | GOMEMLIMIT, GOGC, GOMAXPROCS, GOPATH, GOFLAGS... |
| CmdLine | go binary; .go arg |
| Files | go.mod, go.sum, go.work, *.go |
| Maps | — (statically compiled) |
| ExePath | /go/bin/, golang, /usr/local/go/ |
```

##### Deployment Skills (Used by Phase 2)

Each targets a specific host/VM deployment variant:

| Skill | Detects | OS | Supervisor |
|-------|---------|----|----|
| `tomcat-linux-systemd/` | Apache Tomcat | Linux | systemd |
| `tomcat-windows-service/` | Apache Tomcat | Windows | SC Manager |
| `springboot-jar-linux-systemd/` | Spring Boot JAR | Linux | systemd |
| `springboot-jar-windows-service/` | Spring Boot JAR | Windows | NSSM/WinSW |
| `aspnetcore-linux-nginx-systemd/` | ASP.NET Core | Linux (nginx) | systemd |
| `aspnetcore-windows-iis/` | ASP.NET Core | Windows | IIS |
| `jboss-wildfly-linux-systemd/` | JBoss WildFly | Linux | systemd |

**Skill.md structure:**
```yaml
---
name: tomcat-linux-systemd
description: Apache Tomcat on Linux as systemd service
---

## Detection

Process signals for `motadata-host-agent` to identify this deployment:
- Environment: CATALINA_HOME, CATALINA_BASE
- Command: java process with CATALINA_HOME env var
- Parent: systemd (not init, not standalone)
- Config file: /etc/systemd/user/tomcat.service

## Provisioning

Infrastructure setup (infra-setup agent):
- Install Java runtime
- Download Tomcat
- Create systemd unit file

Application setup (app-setup agent):
- Deploy WAR/JAR to webapps/
- Start service
- Verify /health endpoint
```

---

## Data Flow: From Command to Binary

### Example: `/start-code java`

```
User types: /start-code java
    ↓
.claude/commands/start-code.md
    ├─ Validates "java" as real language
    ├─ Points to add-detect-language-code.md
    └─ Invokes agent
    ↓
.claude/agents/add-detect-language-code.md
    ├─ Parse argument: "java"
    ├─ Resolve aliases: none needed
    ├─ Validate: check LANGUAGE_DETECTION_REGISTRY.md row exists ✓
    ├─ Read current coverage from LANGUAGE_DETECTION_REGISTRY.md
    ├─ Read live source: internal/system/language.go
    ├─ Read Signal Catalogue: .claude/skills/detect-language/SKILL.md
    ├─ Compare: find what signals are MISSING
    │   (e.g., maybe missing CATALINA_HOME env var, missing *.ear file marker)
    ├─ Add missing signals to language.go
    │   (edit detectFromEnviron(), detectFromFiles(), etc.)
    ├─ Add test case to language_test.go
    │   (one case per new signal, named "java CATALINA_HOME environ var")
    ├─ Run gofmt ./internal/system/
    ├─ Run go test ./internal/system/... ✓ PASS
    ├─ Run go build -o bin/motadata-host-agent
    ├─ Update LANGUAGE_DETECTION_REGISTRY.md
    │   (increment signal count, add changelog entry)
    ├─ Update CLAUDE.md
    │   (sync coverage table, add agent change log entry)
    └─ Print summary report
    ↓
Output:
    Language:       java
    Signals added:  12 (environ×3, cmdline×1, files×2, maps×2, exepath×4)
    Tests added:    12 new cases
    Test result:    PASS (147 tests)
    Executable:     bin/motadata-host-agent (8.2M)
    LANGUAGE_DETECTION_REGISTRY.md: updated
    CLAUDE.md:      updated
```

---

## Complete Command Lifecycle

### Phase 1: /start-research go

```
/start-research go
    ↓
start-research.md command
    ↓
deployment-researcher agent
    ├─ Research: "How are Go applications deployed on Linux/Windows?"
    ├─ Identify variants:
    │   • go binary as systemd service (Linux)
    │   • go binary as Windows service (NSSM/WinSW)
    │   • Docker container (skip, out of scope)
    ├─ For each variant, create SKILL.md:
    │   .claude/skills/go-linux-systemd/SKILL.md
    │   .claude/skills/go-windows-service/SKILL.md
    └─ Output: detection signals per deployment variant
    ↓
Feeds Phase 2
```

### Phase 2: /start-environment-setup

```
/start-environment-setup
    ↓
start-environment-setup.md command
    ↓
environment-setup agent (Phase 2 orchestrator)
    ├─ Phase 1: Analysis
    │  ├─ Detect OS (uname, /etc/os-release)
    │  ├─ Scan .claude/skills/ for compatible skills
    │  ├─ Cross-reference: skill description vs detected OS
    │  ├─ Check already-running apps (systemctl, curl /health)
    │  └─ Print analysis report (To Provision / Already Deployed / Incompatible)
    │
    ├─ [GATE 1] Await user reply: go / skip / cancel
    │
    ├─ Phase 2: Setup (only if user says "go")
    │  └─ For each To-Provision skill:
    │     ├─ Spawn environment-infra-setup agent
    │     │  └─ Install runtimes (Java, .NET, etc.)
    │     ├─ Spawn environment-app-setup agent
    │     │  └─ Deploy test application, expose /health endpoint
    │     └─ Write docs/environment/current-environment.md
    │
    └─ Phase 3: Start All Apps
       ├─ Read docs/environment/current-environment.md
       ├─ Check systemctl status for each deployed unit
       ├─ Print app status table
       │
       ├─ [GATE 2] If any stopped: await user reply: start / skip
       │
       └─ Start all stopped units via systemctl
    ↓
Output: docs/environment/current-environment.md (app manifest with expected language)
Feeds Phase 3 (ground truth for test validation)
```

### Phase 3: /start-code python

```
/start-code python
    ↓
start-code.md command → add-detect-language-code agent
    ├─ Read .claude/skills/detect-language/SKILL.md
    │  Find Python signals:
    │  ├─ Environ: PYTHONPATH, PYTHONHOME, VIRTUAL_ENV, CONDA_*, PYENV_*
    │  ├─ CmdLine: python, python2, python3; *.py arg; gunicorn, uvicorn, celery
    │  ├─ Files: requirements.txt, setup.py, pyproject.toml, *.py
    │  ├─ Maps: libpython, libpython3
    │  └─ ExePath: python, pyenv, /.pyenv/, /conda/, /anaconda/
    │
    ├─ Read LANGUAGE_DETECTION_REGISTRY.md
    │  Check: which signals are already implemented?
    │
    ├─ Read internal/system/language.go
    │  Check live source for existing Python signals
    │
    ├─ Identify gaps (e.g., missing PIPENV_ACTIVE environ var)
    │
    ├─ Edit internal/system/language.go
    │  └─ Add detectFromEnviron() → case "PIPENV_ACTIVE"
    │
    ├─ Edit internal/system/language_test.go
    │  └─ Add test: "python PIPENV_ACTIVE environ var"
    │
    ├─ Build pipeline:
    │  ├─ gofmt ./internal/system/
    │  ├─ go test ./internal/system/... ✓ PASS (156 tests)
    │  └─ go build -o bin/motadata-host-agent ✓ success (8.3M)
    │
    └─ Update registries
       ├─ LANGUAGE_DETECTION_REGISTRY.md (Python row: add 1 signal, update changelog)
       └─ CLAUDE.md (coverage table, agent change log)
    ↓
Output: Updated bin/motadata-host-agent binary
Feeds Phase 4
```

### Phase 4: /start-test

```
/start-test
    ↓
start-test.md command → test-language-detection agent
    ├─ Read docs/environment/current-environment.md
    │  ├─ Extract expected language per deployed app (from Phase 2)
    │  └─ Example: { app: tomcat, expected_language: "java", pid: 5432, ... }
    │
    ├─ Run bin/motadata-host-agent
    │  └─ JSON output with all detected processes + language
    │
    ├─ For each deployed app:
    │  ├─ Find it in agent output by PID/cmdline
    │  ├─ Compare detected_language vs expected_language
    │  └─ PASS if match, FAIL if mismatch
    │
    ├─ For each FAIL:
    │  ├─ Identify which detection stage missed
    │  │  (environ → cmdline → files → maps → exepath)
    │  └─ Report which stage to target in next /start-code run
    │
    └─ Print summary report
       ├─ PASS: language was detected correctly
       └─ FAIL: stage that failed + feedback for Phase 3 revision
    ↓
Output: Test report (PASS/FAIL per app)
Next action: If FAIL, return to Phase 1 or Phase 3 to add missing signals
```

---

## File Organization Summary

```
.claude/
├── ARCHITECTURE.md              ← This file
├── settings.local.json          ← Permission allowlist for Claude Code harness
│
├── commands/
│   ├── start-code.md            → add-detect-language-code agent
│   ├── start-environment-setup.md → environment-setup agent
│   ├── start-test.md            → test-language-detection agent
│   ├── start-research.md        → deployment-researcher agent
│   └── list-languages.md        → language-lister agent
│
├── agents/
│   ├── add-detect-language-code.md       (Phase 3 execution)
│   ├── environment-setup.md              (Phase 2 execution)
│   ├── test-language-detection.md        (Phase 4 execution)
│   ├── deployment-researcher.md          (Phase 1 execution)
│   └── language-lister.md                (reporting)
│
├── skills/
│   ├── detect-language/
│   │   └── SKILL.md              ← Signal catalogue for all languages
│   │
│   ├── [Deployment Skills — Used by Phase 2]
│   │
│   ├── Linux + systemd
│   │   ├── tomcat-linux-systemd/SKILL.md
│   │   ├── springboot-jar-linux-systemd/SKILL.md
│   │   ├── jboss-wildfly-linux-systemd/SKILL.md
│   │   ├── jetty-linux-systemd/SKILL.md
│   │   ├── aspnetcore-linux-systemd/SKILL.md
│   │   ├── aspnetcore-linux-nginx-systemd/SKILL.md
│   │   ├── aspnetcore-linux-apache-systemd/SKILL.md
│   │   ├── dotnet-worker-linux-systemd/SKILL.md
│   │   └── dotnet-selfcontained-linux-systemd/SKILL.md
│   │
│   ├── Linux + standalone (no supervisor)
│   │   └── tomcat-linux-standalone/SKILL.md
│   │
│   ├── Windows + Service Control Manager
│   │   ├── tomcat-windows-service/SKILL.md
│   │   ├── springboot-jar-windows-service/SKILL.md
│   │   ├── jboss-wildfly-windows-service/SKILL.md
│   │   ├── aspnetcore-windows-service/SKILL.md
│   │   └── dotnet-worker-windows-service/SKILL.md
│   │
│   ├── Windows + IIS
│   │   ├── aspnetcore-windows-iis/SKILL.md
│   │   ├── aspnet-framework-windows-iis/SKILL.md
│   │   └── dotnet-framework-windows-service/SKILL.md
│   │
│   └── [Internal Helper Skills — Used by environment-setup agent]
│       ├── environment-setup/SKILL.md
│       ├── environment-infra-setup/SKILL.md
│       ├── environment-app-setup/SKILL.md
│       └── environment-setup-troubleshooter/SKILL.md
│
└── logs/
    └── environment-setup-*.md   ← Execution logs from Phase 2
```

---

## Key Design Principles

### 1. Separation of Concerns
- **Commands:** Thin validation + delegation (what the user types)
- **Agents:** Detailed execution logic (how work gets done)
- **Skills:** Reusable patterns (what knowledge is captured)

### 2. Deterministic Pipeline
- Phases run in order: 1 → 2 → 3 → 4
- Gates enforce user approval at decision points
- Early validation saves tokens (check `LANGUAGE_DETECTION_REGISTRY.md` before reading 3 files)

### 3. Single Source of Truth
- Live code in `internal/system/language.go` is authoritative (agents grep it, never hardcode)
- `LANGUAGE_DETECTION_REGISTRY.md` is updated BEFORE `CLAUDE.md` (so coverage reports are always accurate even mid-run)
- Signal Catalogue in `skills/detect-language/SKILL.md` is the baseline for all additions

### 4. Fail-Safe Execution
- Build pipeline stops on first failure (gofmt → test → build)
- Errors are printed in full; agent fixes root cause and retries
- No silent partial success; all-or-nothing semantics

### 5. Reusability at Scale
- Each deployment skill is independent (Linux + Java + systemd ≠ Windows + Java + SC Manager)
- Skills can be written by research agents (Phase 1) and reused by setup agents (Phase 2)
- One signal catalogue serves all detection agents

---

## When to Edit Each File Type

| File Type | Edit When | Responsibility |
|-----------|-----------|-----------------|
| Command (`.claude/commands/*.md`) | Adding new slash command or changing command signature | Command owner; should be rare |
| Agent (`.claude/agents/*.md`) | Changing execution logic, procedure, or validation rules | Agent owner; updated as project scales |
| Skill (`.claude/skills/**/SKILL.md`) | Adding signals for a language or documenting a new deployment variant | Research agents (Phase 1) or manual curation |
| LANGUAGE_DETECTION_REGISTRY.md | Tracks coverage per language (auto-updated by add-detect-language-code agent) | Agents; never manually edit |
| CLAUDE.md | Project context, coverage snapshot, change log (auto-updated by agents) | Agents; never manually edit |
| settings.local.json | Granting permissions to tools, setting env vars | User; via `/update-config` skill |

---

## Debugging Checklist

**Agent fails early (validation)?**
- Check command validation rules in `.claude/commands/<cmd>.md`
- Check startup protocol in agent file (e.g., "Validate early, save tokens")

**Build fails (gofmt, test, or compile)?**
- Agent prints exact error + location
- Check that signals are not duplicated (grep `language.go` for the string)
- Check that test name follows convention: `"<lang> <signal description>"`

**Test suite fails (Phase 4)?**
- Check `docs/environment/current-environment.md` exists and is valid (Phase 2 output)
- Check app is actually running: `systemctl --user is-active <unit>`
- Re-run Phase 3 (`/start-code`) to target the failed detection stage

**Already-deployed app not recognized (Phase 2)?**
- Skill description must mention the detected OS/distro
- App must be listening on expected port OR service must be active
- Check `/health` endpoint responds with 200 (Phase 2 checks this)

---

## Quick Reference: What Gets Updated When

```
/start-code <lang>
    ↓
    internal/system/language.go          (signal additions)
    internal/system/language_test.go     (test cases)
    LANGUAGE_DETECTION_REGISTRY.md       (coverage, changelog)
    CLAUDE.md                            (coverage table, agent changelog)
    bin/motadata-host-agent              (rebuilt binary)

/start-environment-setup
    ↓
    docs/environment/current-environment.md  (app manifest)
    .claude/logs/environment-setup-*.md      (execution log)

/start-test
    ↓
    (no files updated; PASS/FAIL report printed)

/start-research <lang>
    ↓
    .claude/skills/<deployment>/SKILL.md     (new skills created)
    CLAUDE.md                               (change log entry)
```

---

## Common Workflows

### Add signals for a new language
```bash
/start-code kotlin
# Reads from skills/detect-language/SKILL.md → Kotlin section
# Adds signals to language.go and language_test.go
# Updates registries
# Rebuilds binary
```

### Stand up a test environment
```bash
/start-environment-setup
# Analyzes OS
# Provisions compatible apps using skills/
# Writes docs/environment/current-environment.md
```

### Validate detection against deployed app
```bash
/start-test
# Runs binary against deployed app
# Compares detected language vs expected language
# Reports PASS or which stage to fix in next /start-code
```

### Research how a language is deployed on a new OS variant
```bash
/start-research go
# Researches "How is Go deployed on [OS]?"
# Creates .claude/skills/go-[os]-[supervisor]/SKILL.md
# These skills then feed Phase 2 provisioning
```

