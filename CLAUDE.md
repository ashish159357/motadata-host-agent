# motadata-host-agent — Project Context

> Loaded automatically by Claude Code at every session start.
> Updated by agents after every successful run. Do not delete or rename this file.
>
> **Agent commands:** `/start-code` (language detection coding) · `/detect-language <lang>` · `/list-languages`

## Purpose

Bare-metal service discovery agent. Scans all running processes on a Linux host
via the `/proc` filesystem, identifies the programming language of each service,
and reports results as structured JSON (optionally POSTing to a Motadata server).

Replaces the Kubernetes-specific `motadata-agent` with a host-level equivalent
that needs no cluster access. Do not add Kubernetes-specific dependencies or
assumptions back into this codebase.

## Module

```
github.com/motadata/motadata-host-agent   (Go 1.24)
```

## Directory Structure

```
motadata-host-agent/
├── CLAUDE.md                              ← this file (agent reads + updates every run)
├── README.md                              ← user-facing documentation
├── go.mod
├── Dockerfile
│
├── .claude/
│   ├── settings.local.json               ← Claude Code permission allowlist
│   ├── agents/
│   │   ├── code-generator.md             ← /start-code agent logic
│   │   └── language-lister.md            ← /list-languages agent logic
│   ├── commands/
│   │   ├── start-code.md                 ← /start-code entry point
│   │   └── list-languages.md             ← /list-languages entry point
│   └── skills/
│       └── detect-language/
│           └── SKILL.md                  ← /detect-language <lang> agent definition
│
├── cmd/motadata-host-agent/
│   └── main.go                           ← binary entry point
│
├── internal/
│   ├── config/config.go                  ← env-var configuration (Config struct)
│   ├── agent/service.go                  ← periodic discovery + HTTP POST loop
│   │
│   ├── host/                             ← structured package with confidence scoring
│   │   ├── types.go                      ← ProcessInfo, Detection, ServiceRecord
│   │   ├── scanner.go                    ← /proc scanner (ProcRoot configurable)
│   │   ├── scanner_test.go
│   │   ├── language.go                   ← language detection with confidence levels
│   │   └── language_test.go
│   │
│   └── system/                           ← 5-stage pipeline package (agent-managed)
│       ├── process.go                    ← DiscoverServices, ServiceCache, processInfo
│       ├── language.go                   ← DetectLanguage (environ→cmdline→files→maps→exe)
│       └── language_test.go              ← table-driven unit tests
│
└── bin/motadata-host-agent               ← compiled binary (ELF x86-64)
```

## Agent Commands

| Command | What it does |
|---------|--------------|
| `/start-code` | Closes coverage gaps in `language.go` + tests, rebuilds binary |
| `/detect-language <lang>` | Adds missing signals for one language, updates `LANGUAGES.md` |
| `/list-languages` | Reports coverage per language from `LANGUAGES.md` |

## Pipeline Architecture

The project is driven by a 4-phase pipeline. Each phase is invoked manually
through a slash command in `.claude/commands/`, and each phase's output feeds
the next.

```
┌──────────────────┐    ┌──────────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│ Phase 1          │    │ Phase 2              │    │ Phase 3          │    │ Phase 4          │
│ /start-research  │───▶│ /start-environment-  │───▶│ /start-code      │───▶│ /start-test      │
│                  │    │  setup               │    │                  │    │                  │
│ Research host/VM │    │ Provision host and   │    │ Close detection  │    │ Run agent binary │
│ deployment       │    │ deploy a real        │    │ gaps in          │    │ against deployed │
│ patterns per     │    │ application using    │    │ internal/system, │    │ app; verify it   │
│ language.        │    │ Phase 1 skills.      │    │ rebuild binary.  │    │ detects language.│
│                  │    │                      │    │                  │    │                  │
│ Output:          │    │ Output:              │    │ Output:          │    │ Output:          │
│ .claude/skills/  │    │ docs/environment/    │    │ bin/motadata-    │    │ PASS/FAIL report │
│ *-SKILL.md       │    │ current-             │    │ host-agent       │    │ with reasons for │
│ (deployment      │    │ environment.md       │    │ (ELF binary)     │    │ any miss.        │
│ detection        │    │                      │    │                  │    │                  │
│ skills)          │    │                      │    │                  │    │                  │
└──────────────────┘    └──────────────────────┘    └──────────────────┘    └──────────────────┘
```

### Phase 1 — `/start-research`
- **Goal:** Research how a given language's services are deployed on real
  hosts/VMs (systemd, init scripts, container runtimes, IIS, Tomcat, etc.).
- **Produces:** One or more `SKILL.md` files under `.claude/skills/` that encode
  signals used by Phase 2 to provision and by Phase 3 to detect.
- **Consumed by:** Phase 2 (environment setup) and Phase 3 (detection coding).

### Phase 2 — `/start-environment-setup` *(planned, not yet implemented)*
- **Goal:** Using the skills from Phase 1, stand up a real deployment of a
  chosen application on the current host/VM.
- **Produces:** `docs/environment/current-environment.md` describing:
  - which environment was provisioned (OS, runtime, supervisor),
  - which application is deployed and how (binary path, service unit, ports),
  - expected language label the agent should emit.
- **Consumed by:** Phase 4 (ground truth for PASS/FAIL).

### Phase 3 — `/start-code`
- **Goal:** Close missing detection signals in `internal/system/language.go`
  and its tests, then rebuild.
- **Produces:** Updated `internal/system/` sources, refreshed tests, and a
  rebuilt `bin/motadata-host-agent`.
- **Consumed by:** Phase 4 (binary under test).

### Phase 4 — `/start-test`
- **Goal:** Run the freshly built agent against the environment described in
  `docs/environment/current-environment.md` and confirm that every deployed
  application is reported with the correct `language`.
- **Produces:** A PASS/FAIL report. On FAIL, the report lists the specific
  signal stage (environ → cmdline → files → maps → exepath) that missed, so the
  next `/start-code` run can target it.
- **Feedback loop:** FAIL results motivate a new Phase 1 / Phase 3 iteration.

## Package Responsibilities

| Package | Role |
|---------|------|
| `internal/config` | Loads env vars into `Config`; applies defaults |
| `internal/agent` | Orchestration: discovery loop + HTTP POST to server |
| `internal/host` | Full-featured scanner with `ProcessInfo`, `Detection`, confidence scoring |
| `internal/system` | Lightweight 5-stage pipeline; target of `/detect-language` agent edits |

## Coding Conventions

- **Formatting:** always run `gofmt`; keep imports sorted.
- **Naming:** exported identifiers `PascalCase`, internal helpers `camelCase`, tests `TestXxx`.
- **Style:** prefer small focused helpers over large monolithic functions; keep detection heuristics deterministic and easy to read.
- **Scope:** only edit files required for the requested change; preserve existing behaviour outside that scope.
- **Comments:** add a comment only when the *why* is non-obvious — never describe *what* the code does.

## Detection Rules

Discovery reads from `/proc` only (no external calls, no Kubernetes API).
Language signals are evaluated in descending confidence order:

1. Command-line interpreter name or script file extension (highest confidence)
2. Environment variable hints set by the runtime
3. Working-directory file markers (e.g. `package.json`, `go.mod`)
4. Shared-library names from `/proc/[pid]/maps`
5. Executable install-path heuristics (lowest confidence)

Unknown processes must remain `""` — never guess if no signal matches.

## Security & Runtime Notes

- Do not log or store sensitive environment values beyond what detection requires.
- Do not assume root privileges; tolerate unreadable `/proc` entries and continue scanning.
- Process state can change mid-scan — handle missing files gracefully with early returns.

## Language Detection — internal/system pipeline

`DetectLanguage(proc processInfo) string` in `internal/system/language.go`
runs five stages in priority order, returning at the first match:

| # | Stage | Data source | Function |
|---|-------|-------------|----------|
| 1 | **Environ** | `/proc/[pid]/environ` | `detectFromEnviron` |
| 2 | **CmdLine** | `/proc/[pid]/cmdline` | `detectFromCmdLine` |
| 3 | **Files** | `readdir(/proc/[pid]/cwd)` | `detectFromFiles → classifyByFiles` |
| 4 | **Maps** | `/proc/[pid]/maps` | `detectFromMaps` |
| 5 | **ExePath** | `readlink(/proc/[pid]/exe)` | `detectFromExecutablePath` |

## Supported Languages & Coverage (internal/system)

<!-- AGENT-MANAGED: the /detect-language agent updates this table after every run -->

| Language | Environ signals | CmdLine signals | File markers | Maps signals | ExePath signals | Last updated |
|----------|----------------|-----------------|--------------|--------------|-----------------|--------------|
| go | GOMEMLIMIT, GOGC, GOMAXPROCS, GOPATH, GOFLAGS, GONOSUMDB, GONOSUMCHECK | `go` binary; `.go` arg | go.mod, go.sum, go.work, go.work.sum, *.go | — (static) | /go/bin/, golang, /usr/local/go/ | 2026-04-17 |
| java | JAVA_TOOL_OPTIONS, JAVA_OPTS, CATALINA_*, JVM_OPTS, JAVA_HOME, JAVA_VERSION, CLASSPATH, JDK_JAVA_OPTIONS | java, javaw; -jar flag; *.jar arg | pom.xml, build.gradle, build.gradle.kts, settings.gradle, gradlew, *.jar, *.war, *.ear | libjvm.so, libjava.so | jdk, jre, openjdk, java- | 2026-04-18 |
| python | PYTHONPATH, PYTHONHOME, VIRTUAL_ENV, CONDA_* | python, python2, python3; *.py | requirements.txt, setup.py, pyproject.toml, *.py | libpython | python, pyenv | — |
| nodejs | NODE_ENV, NODE_OPTIONS, NODE_PATH, NPM_CONFIG_PREFIX | node, nodejs, npm, npx, yarn, pnpm; *.js, *.mjs | package.json, node_modules | libnode.so | nodejs, /node/bin/ | — |
| dotnet | ASPNETCORE_*, DOTNET_* | dotnet | appsettings.json, web.config, *.pdb, *.deps.json | libcoreclr, libmono | dotnet | — |
| ruby | RAILS_ENV, RACK_ENV, BUNDLE_PATH | ruby, bundle; *.rb | Gemfile, Gemfile.lock, *.rb | libruby | ruby, rbenv, /rvm/ | — |
| php | PHP_INI_SCAN_DIR, PHPRC | php, php-fpm; *.php | composer.json, composer.lock, *.php | libphp | /php/, phpenv | — |
| perl | — | perl, perl5 | — | — | — | — |
| rust | CARGO_HOME, RUSTUP_HOME | — | cargo.toml, cargo.lock | — (static) | rustup, /.cargo/ | — |
| cpp | — | — | CMakeLists.txt + *.cpp | libstdc++.so | — | — |

## Build & Test Commands

Run from the project root (`motadata-host-agent/`):

```bash
# Build binary
GOROOT=/snap/go/current bash -c 'CGO_ENABLED=0 go build -o bin/motadata-host-agent ./cmd/motadata-host-agent/'

# Run a single scan and print JSON to stdout
GOROOT=/snap/go/current bash -c 'go run ./cmd/motadata-host-agent'

# Write scan output to a file
GOROOT=/snap/go/current bash -c 'OUTPUT_FILE=/tmp/motadata-host-scan.json go run ./cmd/motadata-host-agent'

# Test all packages
GOROOT=/snap/go/current bash -c 'go test ./... -v'

# Test agent-managed package only
GOROOT=/snap/go/current bash -c 'go test ./internal/system/... -v'

# Format agent-managed package
GOROOT=/snap/go/current bash -c 'gofmt -w ./internal/system/'
```

> If the local Go toolchain needs an explicit GOROOT, set it only for the command
> being run (as shown above) rather than changing project files.

## Testing Guidelines

- Use table-driven tests for all detection logic and scan behaviour.
- Favour fake `/proc` fixtures and temporary directories over system-dependent tests.
- Every test run must verify: multiple processes detected in one scan, correct language
  inference for common stacks, and that unknown processes remain `""`.
- Run `go test ./...` before any change that touches scanning or detection logic.

## Runtime Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `MOTADATA_SERVER_URL` | `""` | HTTP endpoint for POST reports |
| `HOST_NAME` | `os.Hostname()` | Host identifier in the payload |
| `DEPLOYMENT` | `""` | Deployment label |
| `POST_INTERVAL_SECONDS` | `60` | Discovery + report cycle interval |

## JSON Output Shape

```json
{
  "host.name": "prod-host-01",
  "deployment": "production",
  "services": [
    {
      "name": "java",
      "pid": 1234,
      "executable": "/usr/lib/jvm/java-11/bin/java",
      "cmdline": "java -jar /opt/myapp/app.jar",
      "working_dir": "/opt/myapp",
      "instances": 1,
      "language": "java",
      "user": "1001"
    }
  ]
}
```

## Commit & PR Guidance

- Use short imperative commit subjects: `Add proc-based language detection`, `Extend Go environ signals`.
- Keep changes focused — one logical change per commit.
- When reviewing, prioritise correctness of heuristics, failure handling, and test coverage over style-only changes.

---

## Agent Change Log

<!-- AGENT-MANAGED: /detect-language prepends a row here after every successful run -->

| Date | Command | Language | Signals added | Tests added |
|------|---------|----------|---------------|-------------|
| 2026-04-18 | /start-code java | java | environ×9, cmdline×2, files×8, maps×2, exepath×4 | 15 |
| 2026-04-17 | /detect-language java | java | environ×3, cmdline×1, files×2, exepath×1 | 8 |
| 2026-04-17 | /detect-language go | go | environ×4, cmdline×1, files×3, exepath×1 | 7 |
