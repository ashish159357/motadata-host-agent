# /start-code — Phase 3

Generates and enhances service and language detection code, then builds
the binary. No arguments required.

## What this command does

Phase 3 reads the runtimes discovered in Phase 1 and the coverage gaps
recorded in `CLAUDE.md`, then edits `internal/system/language.go` and
`internal/system/language_test.go` to close those gaps. It formats, tests,
builds, and updates `CLAUDE.md` with the resulting coverage state.

## Before starting

1. Read `CLAUDE.md` from the project root.
2. Verify Phase 1 is present (dated environment analysis). If not:
   > Run `/start-environment-analysis` first.
3. Verify Phase 2 build status is OK. If the Phase 2 section shows
   "FAILED" for the build:
   > Run `/start-environment-setup` to fix the build before generating code.

## Execution

Follow the full instructions in:

```
.claude/agents/code-generator.md
```

The agent decides which languages to enhance based on Phase 1 findings
and the current coverage table. It edits only the two files listed above,
runs format + tests + build, and updates `CLAUDE.md`.

Only files explicitly allowed by the agent spec may be touched:
- `internal/system/language.go`
- `internal/system/language_test.go`
- `CLAUDE.md`

## After completion

Tell the user:
- Which languages were enhanced and how many signals were added.
- Total test count and pass/fail result.
- Binary location and size.
- What to run next: `/start-testing`
