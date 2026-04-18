# /start-code <lang>

Enhances language detection for one specific language in
`internal/system/language.go`. Adds missing signals across all five
detection stages (environ → cmdline → files → maps → exepath), writes
matching tests, runs gofmt + tests + build, then updates `LANGUAGE_DETECTION_REGISTRY.md`
and `CLAUDE.md`.

## Usage

```
/start-code <lang>
```

Accepts any real programming language. Examples:

```
/start-code assembly
/start-code kotlin
/start-code swift
/start-code typescript
/start-code perl
/start-code rust
/start-code cpp
```

**Aliases accepted:** `node` → `nodejs`, `js` → `nodejs`, `.net`/`csharp`/`c#` → `dotnet`,
`c++` → `cpp`, `objc` → `objectivec`, `ts` → `typescript`, `sh` → `shell`, `asm` → `assembly`

**Invalid/joke languages are rejected immediately** (e.g. `bhailang`, `lolcode` as a service,
random words) — validation happens before any file I/O to avoid wasting tokens.

## Execution

Follow the full instructions in:

```
.claude/agents/add-detect-language-code.md
```

## Validation behaviour

The agent checks the argument against a **Valid Language Registry** (40+ real languages)
before reading any file. If the language is not recognised:

```
"<arg>" is not a recognised programming language.
```

No files are touched. This keeps token usage low for invalid inputs.

## What it does (for valid languages)

1. Reads `LANGUAGE_DETECTION_REGISTRY.md` and `CLAUDE.md` to determine what signals are already present
2. Cross-references against the Signal Catalogue to find gaps
3. Edits `internal/system/language.go` — inserts missing signals into the correct stage functions
4. Edits `internal/system/language_test.go` — adds one table-driven test case per new signal
5. Runs `gofmt`, `go test ./internal/system/...`, and `go build` — stops on first failure
6. Updates `LANGUAGE_DETECTION_REGISTRY.md` coverage row and change log
7. Updates `CLAUDE.md` coverage table and Agent Change Log

## Signal coverage

- **Core languages** (go, java, python, nodejs, dotnet, ruby, php, perl, rust, cpp):
  use detailed pre-built signal catalogues
- **Extended languages** (assembly, kotlin, swift, typescript, dart, elixir, haskell, etc.):
  use starter catalogues + agent inference
- **Other registry languages** (ada, cobol, fortran, vhdl, etc.):
  agent infers signals from language knowledge

## Files touched

- `internal/system/language.go`
- `internal/system/language_test.go`
- `LANGUAGE_DETECTION_REGISTRY.md`
- `CLAUDE.md`

## After completion

Prints a summary report:

```
Language:         <lang>
Signals added:    <N> (environ×A, cmdline×B, files×C, maps×D, exepath×E)
Tests added:      <N> new cases
Test result:      PASS (<total> tests)
Executable:       bin/motadata-host-agent  (<size>)
LANGUAGE_DETECTION_REGISTRY.md:     updated
CLAUDE.md:        updated
```
