# /list-languages

Lists all languages that currently have detection code in `internal/system/language.go`.
No arguments required.

## What this command does

Reads the live source file (`language.go`) and the coverage table in `CLAUDE.md`,
then prints a formatted report showing:

- Which languages are **fully or partially implemented** (with a per-stage breakdown)
- Which languages are in the supported list but **not yet implemented**

This is a read-only command — it never modifies any file.

## Execution

Follow the full instructions in:

```
.claude/agents/language-lister.md
```

## After completion

Tell the user:
- How many languages are implemented vs. pending.
- Which stages (environ / cmdline / files / maps / exepath) each implemented language covers.
- Which language to target next with `/detect-language <lang>` if coverage is incomplete.
