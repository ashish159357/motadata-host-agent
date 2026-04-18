---
description: Close language detection gaps in internal/system for one language
argument-hint: <language>
---

The user wants to enhance language detection for: **$ARGUMENTS**

Delegate the work to the `add-detect-language-code` sub-agent (defined at `.claude/agents/add-detect-language-code.md`), which owns the gap analysis, code edits, tests, build verification, and documentation updates for `internal/system/language.go`.

## Workflow

1. Normalise `$ARGUMENTS`: lowercase and trim. Resolve aliases: `node` → `nodejs`, `.net`/`csharp` → `dotnet`, `c++` → `cpp`.
2. If the resolved language is not one of `go, java, python, nodejs, dotnet, ruby, php, perl, rust, cpp`, print:
   > `Language "<arg>" is not supported. Supported: go, java, python, nodejs, dotnet, ruby, php, perl, rust, cpp`
   and stop. Do not spawn the sub-agent.
3. Spawn **one** `Agent` tool call with `subagent_type: "add-detect-language-code"`. The prompt must include:
   - `LANG = <resolved language>`
   - Instruction: *"Follow your agent definition. Perform gap analysis against LANGUAGES.md and the Reference Signal Catalogue, edit `internal/system/language.go` and `internal/system/language_test.go`, run gofmt + tests + build, then update LANGUAGES.md and CLAUDE.md."*
4. After the sub-agent returns, relay its final completion report (Language, Signals added, Tests added, Test result, Executable, LANGUAGES.md, CLAUDE.md) to the user verbatim.
5. If the sub-agent reports a build or test failure, surface the exact error output to the user and stop — do not retry blindly.

## Rules

- Do not edit `language.go`, `language_test.go`, `LANGUAGES.md`, or the coverage table in `CLAUDE.md` yourself. Those files are owned by the sub-agent.
- Do not spawn the sub-agent for an unsupported language.
- Run exactly one sub-agent per invocation.
