# Agent: code-generator

Generates or enhances service and language detection code in
`internal/system/language.go` and `internal/system/process.go` based on the
runtimes discovered in Phase 1. Writes status into the **Phase 3** section of
`CLAUDE.md`.

---

## Inputs

- `CLAUDE.md` — read Phases 1 and 2 first.
  - If Phase 1 is missing: stop, ask user to run `/start-environment-analysis`.
  - If Phase 2 build status is FAILED: stop, ask user to run `/start-environment-setup`.
- `internal/system/language.go` — current detection logic.
- `internal/system/language_test.go` — current tests.
- The "Supported Languages & Coverage" table in `CLAUDE.md`.

---

## Detection pipeline reference

Five stages in priority order (do not reorder):

| # | Stage | Source | Function |
|---|-------|--------|----------|
| 1 | Environ | `/proc/[pid]/environ` | `detectFromEnviron` |
| 2 | CmdLine | `/proc/[pid]/cmdline` | `detectFromCmdLine` |
| 3 | Files | cwd listing | `classifyByFiles` |
| 4 | Maps | `/proc/[pid]/maps` | `detectFromMaps` |
| 5 | ExePath | exe symlink | `detectFromExecutablePath` |

---

## Steps

### 1. Read current coverage from CLAUDE.md

Parse the "Supported Languages & Coverage" table. For each language, record
which stages already have signals and which are empty (`—`).

### 2. Determine target languages

Cross-reference with Phase 1's "Installed runtimes" table. Prioritise
languages that are:
1. Installed on this host AND have gaps in coverage.
2. Installed and have no coverage at all.
3. Not installed but referenced in the detection table.

### 3. Enhance detection for each target language

For each language with gaps, apply the signal catalogue below. Only add
signals that are absent — never duplicate.

#### Signal catalogue

**java**
- Environ: `JAVA_TOOL_OPTIONS`, `JAVA_OPTS`, `CATALINA_*`, `JVM_OPTS`, `JAVA_HOME`, `JAVA_VERSION`, `CLASSPATH`, `JDK_JAVA_OPTIONS`
- CmdLine: `java`, `javaw`; `-jar` flag; `.jar` arg extension
- Files: `pom.xml`, `build.gradle`, `build.gradle.kts`, `settings.gradle`, `gradlew`, `*.jar`, `*.war`, `*.ear`
- Maps: `libjvm.so`, `libjava.so`
- ExePath: `jdk`, `jre`, `openjdk`, `java-`

**python**
- Environ: `PYTHONPATH`, `PYTHONHOME`, `VIRTUAL_ENV`, `CONDA_PREFIX`, `CONDA_DEFAULT_ENV`, `CONDA_EXE`, `PYENV_VERSION`, `PYENV_ROOT`, `PIPENV_ACTIVE`
- CmdLine: `python`, `python2`, `python3`, `python3.x`; `.py` arg; `gunicorn`, `uvicorn`, `celery`
- Files: `requirements.txt`, `setup.py`, `pyproject.toml`, `setup.cfg`, `Pipfile`, `Pipfile.lock`, `poetry.lock`, `manage.py`, `wsgi.py`, `asgi.py`, `*.py`
- Maps: `libpython`, `libpython3`
- ExePath: `python`, `pyenv`, `/.pyenv/`, `/conda/`, `/miniconda/`, `/anaconda/`

**nodejs**
- Environ: `NODE_ENV`, `NODE_OPTIONS`, `NODE_PATH`, `NPM_CONFIG_PREFIX`, `NVM_DIR`, `NVM_BIN`, `YARN_GLOBAL_FOLDER`, `BUN_INSTALL`
- CmdLine: `node`, `nodejs`, `npm`, `npx`, `yarn`, `pnpm`, `bun`, `deno`; `.js`, `.mjs`, `.cjs` args
- Files: `package.json`, `node_modules`, `.nvmrc`, `.node-version`, `bun.lockb`, `yarn.lock`, `pnpm-lock.yaml`, `deno.json`, `*.js`, `*.mjs`
- Maps: `libnode.so`
- ExePath: `nodejs`, `/node/bin/`, `/.nvm/`, `/bun/`, `/deno/`

**go**
- Environ: `GOMEMLIMIT`, `GOGC`, `GOMAXPROCS`, `GOPATH`, `GOFLAGS`, `GONOSUMDB`, `GONOSUMCHECK`
- CmdLine: `go` binary; `.go` arg extension
- Files: `go.mod`, `go.sum`, `go.work`, `go.work.sum`, `*.go`
- Maps: — (static)
- ExePath: `/go/bin/`, `golang`, `/usr/local/go/`

**dotnet**
- Environ: `ASPNETCORE_*`, `DOTNET_*`
- CmdLine: `dotnet`; `.dll` arg
- Files: `appsettings.json`, `web.config`, `global.json`, `*.pdb`, `*.deps.json`, `*.csproj`, `*.sln`, `*.fsproj`, `*.vbproj`
- Maps: `libcoreclr`, `libmono`, `libhostfxr`
- ExePath: `dotnet`, `/.dotnet/`

**ruby**
- Environ: `RAILS_ENV`, `RACK_ENV`, `BUNDLE_PATH`, `GEM_HOME`, `GEM_PATH`, `RUBY_VERSION`, `RBENV_VERSION`
- CmdLine: `ruby`, `bundle`, `rails`, `rake`, `puma`, `unicorn`, `sidekiq`; `.rb` arg
- Files: `Gemfile`, `Gemfile.lock`, `.ruby-version`, `Rakefile`, `config.ru`, `*.rb`, `*.gemspec`
- Maps: `libruby`, `libruby-static`
- ExePath: `ruby`, `rbenv`, `/rvm/`, `/.rbenv/`

**php**
- Environ: `PHP_INI_SCAN_DIR`, `PHPRC`, `PHP_VERSION`, `COMPOSER_HOME`, `COMPOSER_ALLOW_SUPERUSER`
- CmdLine: `php`, `php-fpm`, versioned `php8.x`/`php7.x`; `.php` arg
- Files: `composer.json`, `composer.lock`, `artisan`, `index.php`, `*.php`
- Maps: `libphp`, `libphp8`, `libphp7`
- ExePath: `/php/`, `phpenv`, `php-fpm`

**perl**
- Environ: `PERLLIB`, `PERL5LIB`, `PERL_MB_OPT`, `PERL_MM_OPT`, `PERL_LOCAL_LIB_ROOT`
- CmdLine: `perl`, `perl5`; `.pl`, `.pm`, `.cgi` args
- Files: `Makefile.PL`, `Build.PL`, `cpanfile`, `*.pl`, `*.pm`, `*.t`
- Maps: `libperl.so`
- ExePath: `/perl/`, `perlenv`, `/usr/bin/perl`

**rust**
- Environ: `CARGO_HOME`, `RUSTUP_HOME`, `RUSTFLAGS`, `RUST_LOG`, `RUST_BACKTRACE`, `CARGO_MANIFEST_DIR`
- CmdLine: `cargo`, `rustc`; `.rs` arg
- Files: `Cargo.toml`, `Cargo.lock`, `rust-toolchain`, `rust-toolchain.toml`, `*.rs`
- Maps: — (static)
- ExePath: `rustup`, `/.cargo/`, `/cargo/`, `rustc`

**cpp**
- Environ: `CFLAGS`, `CXXFLAGS`, `CC`, `CXX`, `CMAKE_PREFIX_PATH`
- CmdLine: `g++`, `clang++`, `gcc`, `clang`; `.cpp`, `.cc`, `.cxx` args
- Files: `CMakeLists.txt`, `meson.build`, `Makefile` (+ `.cpp`/`.cc`), `conanfile.txt`, `conanfile.py`, `*.cpp`, `*.cc`, `*.cxx`, `*.hpp`
- Maps: `libstdc++.so`, `libgcc_s.so`
- ExePath: `g++`, `clang++`, `gcc`, `clang`

### 4. Edit language.go

Make minimal, targeted edits. Preserve the five-stage function order and
gofmt formatting. Never duplicate an existing signal.

### 5. Add tests in language_test.go

For each new signal, add one table-driven test case in the appropriate
existing `Test*` function. Name format: `"<lang> <signal description>"`.
Do not modify existing cases.

### 6. Format, test, build

```bash
GOROOT=/snap/go/current bash -c 'gofmt -w ./internal/system/'
GOROOT=/snap/go/current bash -c 'go test ./internal/system/... -v'
GOROOT=/snap/go/current bash -c 'CGO_ENABLED=0 go build -o bin/motadata-host-agent ./cmd/motadata-host-agent/'
```

Stop on first failure and report the exact error.

---

## Output — write to CLAUDE.md

Update two sections:

1. **"Supported Languages & Coverage"** table — fill in every signal now present.
2. **Phase 3 section** — replace `### Phase 3: Code Status` with:

```
### Phase 3: Code Status
_Last run: <date>_

| Language | Signals added | Tests added | Status |
|----------|---------------|-------------|--------|
| <lang>   | environ×N, cmdline×N, files×N, maps×N, exepath×N | N | ✅ |
...

Build: <OK / FAILED>
Total tests: <N> passing
```

3. **Agent Change Log** — prepend one row per language enhanced.

Print a summary to the conversation.
