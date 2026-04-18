---
description: Research host/VM deployment patterns for a language and write them as skills
argument-hint: <language>
---

The user wants to research every host/VM deployment pattern for the language: **$ARGUMENTS**

## Input Validation & Fuzzy Matching (BEFORE Phase 1)

1. **Validate the input.** The language must be a non-empty string with alphanumeric characters only (no spaces, special chars, etc.). Fail fast if invalid:
   ```
   Error: Invalid language name "$ARGUMENTS".
   Language names must be alphanumeric (no spaces or special characters).
   ```

2. **Apply fuzzy matching** to handle common typos and aliases (helpful suggestions, not restrictions):
   - `node` → `nodejs`
   - `.net`, `csharp`, `c#` → `dotnet`
   - `c++`, `c+` → `cpp`
   - `py` → `python`
   - `rb` → `ruby`
   - `js` → `nodejs`
   - `ts` → `nodejs`
   
   If a match is found, ask the user: *"Did you mean `<canonical-name>`? Reply with the correct language name or type `go` to proceed with `$ARGUMENTS`."*

3. **Proceed with the resolved language name** (either the fuzzy-matched canonical name or the original input if user confirms) and use it for all subsequent operations.

---

You orchestrate the workflow. The actual per-skill research is delegated to the `deployment-researcher` sub-agent (defined at `.claude/agents/deployment-researcher.md`), which has its own system prompt, tool allowlist, and model pinned.

## Phase 1 — Enumerate (you do this inline, using the canonical language name)

1. Read `research_agent/prompts/enumerator.md` for the scope rules and naming convention. Follow them strictly.
2. Using your own knowledge, produce a comprehensive list of every distinct `(technology, OS, variant)` deployment pattern for the **canonical language** (after fuzzy matching) directly on a host or VM. Exclude Docker, Kubernetes, containers, serverless, PaaS.
3. Write the manifest to `research_agent/manifests/<canonical-lang>.yaml` using the YAML shape specified in the enumerator prompt. Every row must have `include: true`.
4. Read the file back and display its contents to the user.
5. Tell the user:
   > "Edit `research_agent/manifests/<canonical-lang>.yaml` — delete rows or set `include: false` on anything you don't want. Reply `go` when ready, `cancel` to abort."

**Stop here and wait for the user's reply.** Do not proceed to phase 2 on your own.

## Phase 2 — Research (only after the user says `go`)

1. Re-read `research_agent/manifests/<canonical-lang>.yaml` (the user may have edited it).
2. Collect every row with `include: true`. Skip the rest.
3. Read `research_agent/schema/skill_template.md` into memory — you will pass this to every sub-agent as the output contract.
4. **Fan out in parallel.** In a single message, spawn one `Agent` tool call per included skill with `subagent_type: "deployment-researcher"`. Each prompt must contain:
   - A clear instruction line: *"Research and write the SKILL.md for the following deployment variant."*
   - The skill name, technology, OS, and variant from the manifest row
   - The literal text of the SKILL.md template (from `research_agent/schema/skill_template.md`) as the output contract
   - The target path: *"Write your output to `.claude/skills/<skill-name>/SKILL.md`."*

   Do **not** re-embed the researcher's method, budget, or quality rules — those live in the agent definition and are loaded automatically when the sub-agent spawns.

5. After all sub-agents report back, read one of the generated `SKILL.md` files to verify the frontmatter and structure match the template. If any are malformed, flag them to the user by skill name.

6. Summarize:
   - Number of SKILL.md files written
   - Their parent directory (`.claude/skills/`)
   - Suggest spot-checking one file before moving on to another language

## Rules

- **Do not edit the manifest yourself.** The user controls what gets researched.
- **Do not proceed past phase 1 without the user saying `go`.**
- **Spawn sub-agents in parallel** — one message with N `Agent` tool calls, not N messages with one each.
- If the user says `cancel`, stop and confirm cancellation.
