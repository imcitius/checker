# AGENTS.md

Context and instructions for AI coding agents working in this Stoneforge workspace.

## Quick Start

| I need...                  | Where to look                              |
| -------------------------- | ------------------------------------------ |
| Project documentation      | Check your project's `docs/` directory   |
| Core type details          | `sf show <id>` or workspace docs         |
| CLI commands               | `sf help` or `sf <command> --help`     |
| Architecture overview      | Check your project's `docs/` directory   |

---

## Core Concepts

### Element Types

- **Core Types**: Task, Message, Document, Entity
- **Collection Types**: Plan, Workflow, Playbook, Channel, Library, Team
- **All inherit from Element** (id, type, timestamps, tags, metadata, createdBy)

### Dual Storage Model

- **SQLite**: Fast queries, indexes, FTS — the **cache**
- **JSONL**: Git-tracked, append-only — the **source of truth**

### Dependencies

- **Blocking types**: `blocks`, `awaits`, `parent-child` — affect task status
- **Non-blocking**: `relates-to`, `mentions`, `references` — informational only
- `blocked` status is **computed** from dependencies, never set directly

### Agent Roles (Orchestrator)

- **Director**: Owns task backlog, spawns workers, makes strategic decisions
- **Worker**: Executes assigned tasks (ephemeral or persistent)
- **Steward**: Handles code merges, documentation scanning and fixes

---

## CLI Usage

```bash
sf task ready         # List ready tasks
sf task blocked       # List blocked tasks
sf show <id>          # Show element details
sf task create --title "..." --priority 3 --type feature
sf dependency add --type=blocks <blocked-id> <blocker-id>
sf task close <id> --reason "..."
sf stats              # View progress stats
```

---

## Critical Gotchas

1. **`blocked` is computed** — Never set `status: 'blocked'` directly; it's derived from dependencies
2. **`blocks` direction** — `sf dependency add --type=blocks A B` means A is blocked BY B (B completes first)
3. **Messages need `contentRef`** — `sendDirectMessage()` requires a `DocumentId`, not raw text
4. **`sortByEffectivePriority()` mutates** — Returns same array reference, modifies in place
5. **SQLite is cache** — JSONL is the source of truth; SQLite can be rebuilt
6. **No auto cycle detection** — `api.addDependency()` doesn't check cycles; use `DependencyService.detectCycle()`
7. **FTS not indexed on import** — After `sf import`, run `sf document reindex` to rebuild search index
8. **`relates-to` is bidirectional** — Query both directions: `getDependencies()` AND `getDependents()`
9. **Closed/tombstone always wins** — In merge conflicts, these statuses take precedence
10. **Dirty tracking** — All mutations through QuarryAPI; never modify SQLite directly

---

## Implementation Guidelines

### Type Safety

- Use branded types: `ElementId`, `TaskId`, `EntityId`, `DocumentId`
- Implement type guards: `isTask()`, `isElement()`, etc.
- Use `asEntityId()`, `asElementId()` casts only at trust boundaries

### Storage Operations

- All mutations through `QuarryAPI` — never modify SQLite directly
- Dirty tracking marks elements for incremental export
- Content hashing enables merge conflict detection

### Testing

- Tests colocated with source: `*.test.ts` next to `*.ts`
- Integration tests use real SQLite (`:memory:` or temp files)

### Error Handling

- Use `StoneforgeError` with appropriate `ErrorCode`
- CLI formats errors based on output mode (standard, verbose, quiet)

---

## Agent Orchestration Overview

The orchestrator manages AI agent lifecycles for multi-agent task execution:

```
Director → creates tasks, assigns priorities → dispatches to Workers
Workers  → execute tasks in git worktrees → update status, handoff
Stewards → merge completed work, documentation scanning and fixes
```

Override built-in agent prompts by placing files in `.stoneforge/prompts/`.

---

## Commit Guidelines

- Create commits after completing features, refactors, or significant changes
- Only commit files you changed
- Use conventional commit format: `feat:`, `fix:`, `chore:`, `docs:`
