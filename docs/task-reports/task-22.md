# Task 22 Report — Command Workspace

## Overview

Task 22 implementation is complete and ready for human acceptance against the approved Codea V1.1 design baseline:

`docs/superpowers/specs/2026-08-26-codea-v1.1-agent-workspace-design.md`

Implementation checkpoint:

`381ba1a9aceac6a8ea3ed42dcd42635b7cacbb25`

Fresh acceptance Gate:

- Workflow: `Task 22 Command Workspace Gates`
- Run ID: `32981152117`
- Source commit: `381ba1a9aceac6a8ea3ed42dcd42635b7cacbb25`
- Result: **PASS**

Task 23–26 have not been started.

## Delivered scope

### Centralized Command Registry

A terminal-independent `tui/internal/command` package now owns command definitions, parsing, filtering, execution outcomes, deterministic errors, and custom Markdown loading.

A command definition owns:

- Name
- Aliases
- Description
- Category
- Usage
- Source
- Action / route
- RequiredCapability
- Agent route
- Availability
- Prompt template where applicable

The Bubble Tea application consumes structured command outcomes instead of growing command-specific vendor branches.

### Task 22 built-ins

Exactly the Task 22 built-ins are registered:

- `/help`
- `/clear`
- `/status`
- `/sessions`
- `/skills`
- `/agents`
- `/cancel`
- `/doctor`

Task 23/24 commands are not implemented or registered early.

### Command Palette

The application supports:

- `/` opening the palette immediately
- live prefix filtering
- Up/Down selection
- Enter execution
- Esc close
- direct full slash-command submission
- modal keyboard isolation while the palette is open

Existing keyboard shortcuts remain active when the palette is closed.

### Unknown command isolation

Unknown slash commands return deterministic `COMMAND_NOT_FOUND` command errors and never become normal model prompts.

### Custom Markdown commands

V1.1 supports exactly the two approved scopes:

- `distribution/commands/*.md`
- `.codea/commands/*.md`

There is no user-personal command directory.

Markdown frontmatter can route a custom command to a Codea Agent, and `$ARGUMENTS` is expanded before the resulting prompt crosses `AgentRuntime`.

Application regression coverage verifies that the expanded prompt and Agent route are actually sent through the Codea-owned `AgentRuntime` path.

### Conflict and controlled namespace policy

Load priority remains:

`Built-in > Enterprise > Project`

Name/alias conflicts fail closed with deterministic `COMMAND_CONFLICT`.

The full approved V1.1 controlled namespace is reserved now so Enterprise/Project Markdown cannot occupy future controlled commands before their owning tasks register them:

- `/model`
- `/compact`
- `/review`
- `/test`
- `/api-doc`
- `/debug`
- `/view`

This is namespace protection only. Their behavior remains owned by Task 23/24.

## Architecture boundaries preserved

Task 22 does not change the `AgentRuntime` interface.

It does not introduce OpenCode vendor DTOs into Application/TUI, does not add a second Runtime abstraction, and does not modify the pinned OpenCode `v1.18.11` boundary.

Existing Windows, pure-intranet, DLP, approval, Runtime lifecycle, and offline Plugin behavior remains outside the command package and unchanged.

## Verification evidence

Fresh Gate `32981152117` executed against exact implementation checkpoint `381ba1a9aceac6a8ea3ed42dcd42635b7cacbb25`.

Results:

- Task 22 focused tests on Linux — **PASS**
- Full `go test ./... -count=1` — **PASS**
- `go vet ./...` — **PASS**
- `go build ./...` — **PASS**
- Windows x64 cross-build — **PASS**
- Task 22 focused tests on Windows — **PASS**

The existing Windows release workflow also independently completed its `windows-installer-smoke` job successfully on the same source commit, including the locked OpenCode `v1.18.11` and offline Plugin startup gates. This is supplemental evidence; Task 22 acceptance is based on the dedicated Task 22 Gate above.

## Acceptance mapping

- `/` opens palette — PASS
- live filtering — PASS
- keyboard selection — PASS
- direct full command input — PASS
- unknown slash command never reaches model — PASS
- palette keys do not leak into chat/other shortcuts — PASS
- enterprise Markdown loading — PASS
- project Markdown loading — PASS
- `$ARGUMENTS` expansion and AgentRuntime routing — PASS
- deterministic fail-closed conflict handling — PASS
- future controlled built-in namespace protection — PASS
- existing shortcuts/general chat regression — PASS
- Windows build and focused behavior — PASS

## Current status

**Task 22: IMPLEMENTATION COMPLETE / AWAITING HUMAN ACCEPTANCE**

Do not start Task 23 until Task 22 is explicitly accepted.
