# PRD: towerctl – Project Control Tower CLI & MCP Server

**Version:** 1.0  
**Author:** Caesario Dito - Consulted with Deepseek Web Chat Expert
**Status:** Draft  
**Target:** Go implementation, Hermes‑agent integration via native CLI + MCP

---

## 1. Executive Summary

`towerctl` is a local, single‑user context management layer for Hermes-agent and other AI agents. It stores projects, snapshots, and priority context in a SQLite database and exposes both a human-friendly CLI and an MCP (Model Context Protocol) server. This keeps transient project state out of the assistant's long-term memory and provides a structured source of truth for daily planning, progress checks, and context parking.

---

## 2. Problem Statement

Managing multiple concurrent workstreams, side hustles, and personal projects leads to:

- **Fragmented state** – you can’t remember where you stopped or what the next action is.
- **Memory pollution** – storing ephemeral project details inside an AI assistant’s memory degrades its reasoning quality over time.
- **High-friction recall** – re-reading long chat logs or notes just to resume work wastes mental energy.

`towerctl` solves this by providing a **dumb structured store** that the assistant can query and update via CLI or MCP tools, while the user can also interact with it directly in the terminal.

---

## 3. Core Principles

1. **Local first** – all data stored on the user’s machine (SQLite, human‑readable exports).
2. **Low friction** – park a context in seconds with an editor template or via natural language (via the agent).
3. **Single source of truth** – the `towerctl` binary is the only writer to the database. Humans write through CLI commands; agents write through MCP tools; both routes share the same validation, migrations, and transactions.
4. **Agent‑first design** – every command that produces output supports `--format json` for machine consumption; an MCP server is provided for seamless integration.
5. **Reliability** – built in Go, with robust error handling, migrations, and thorough testing.

---

## 4. User Stories

- As a **user**, I want to quickly add a new project and its metadata.
- As a **user**, I want to **park** my context at the end of the day so I can resume tomorrow without mental load.
- As a **user**, I want to get a **morning brief** that gives Hermes-agent enough structured state to recommend today’s target.
- As a **user**, I want Hermes-agent to run a progress check whenever it decides one is useful.
- As a **developer** using Hermes‑agent, I want to give the agent tools to read/write `towerctl` state without dealing with raw shell commands.

---

## 5. Functional Requirements

### 5.1 Data Entities

Core tables (see full schema in section 8):

- **projects**
  - `id` (text, unique slug)
  - `name`
  - `type` (work/side_hustle/personal)
  - `status` (active/blocked/paused/done)
  - `priority` (integer 1‑5)
  - `priority_reason` (text)
  - `last_prioritized_at` (timestamp)
  - `deadline` (date)
  - `notes` (text)
  - `created_at`, `updated_at`

- **snapshots** (context parking)
  - `id` (integer)
  - `project_id` (FK)
  - `stopped_where` (text)
  - `what_changed` (text)
  - `next_action` (text)
  - `blockers` (text)
  - `files_links` (text)
  - `need_help` (text)
  - `created_at`

- **schema_version** (migration tracking)

### 5.2 CLI Commands

All commands output results in human‑readable form by default. A `--format json` flag produces machine‑parseable JSON to `stdout`; errors go to `stderr`.

#### Project Management

```bash
towerctl project add <name> [--type work|side_hustle|personal] [--priority <1-5>] [--priority-reason <text>] [--deadline <date>]
towerctl project list [--status active|blocked|paused|done] [--format json]
towerctl project show <id>
towerctl project update <id> [fields...]
```

#### Context Snapshots (Parking)

```bash
towerctl park <project-id>   # opens $EDITOR with a markdown template for the snapshot
towerctl park <project-id> --stopped "..." --next "..." --blocker "..."  # inline quick park
```

The editor template:

```
# Context Park: <project name>

Where I stopped:

What changed since last session:

Next exact action:

Blockers:

Files/links:

Need help from whom:
```

After saving, the CLI parses the headings and stores the snapshot.

#### Daily Rhythm Commands

- **Morning brief**  
  `towerctl morning`  
  Outputs structured data for all active projects, including metadata, priority context, and latest snapshot. Hermes-agent uses this data to recommend today’s target.

- **Check**  
  `towerctl check`  
  Prints a list of questions tailored to each active project, derived from the last snapshot (e.g., “Did you make progress on X?”). Timing is decided by Hermes-agent, not by towerctl.

- **Parking prompt**  
  `towerctl park-day`  
  Simple CLI fallback that asks fixed parking-template questions and saves snapshots. In agent usage, Hermes-agent drives the conversation and calls MCP tools; towerctl only stores structured answers.

#### Retrieval & Summaries

```bash
towerctl summary           # brief status of all active projects
towerctl next              # returns latest next actions from active projects for agent consumption
```

#### Export

```bash
towerctl export [--format markdown]   # exports all project data to ~/.towerctl/exports/
```

Exports are suitable for git backup and human review.

### 5.3 MCP Server

The same binary can run as an MCP server (stdio transport) when invoked as:

```bash
towerctl serve-mcp
```

This mode exposes all read‑oriented commands and `park` / `park-day` as **tools** in accordance with the Model Context Protocol. No separate server process is needed.

The MCP tools will include:

| Tool Name                | Description                                          | Input Parameters                                 |
|--------------------------|------------------------------------------------------|--------------------------------------------------|
| `get_active_project_context` | Get structured active-project state for agent planning | none                                             |
| `get_project_list`       | List projects, optionally filtered by status         | status (optional)                                |
| `get_project_detail`     | Full detail of one project including latest snapshot | project_id                                       |
| `park_context`           | Create a new snapshot for a project and optionally refresh priority context | project_id, stopped_where, next_action, priority (optional), priority_reason (optional), deadline (optional), status (optional), etc. |
| `start_park_day`         | Return active projects that may need parking review  | none                                             |
| `park_day_update`        | Store one parking answer or finalized parking payload | project_id, field_name, value                    |

All tool results are returned as structured JSON objects.

---

## 6. Technical Architecture

### 6.1 Language & Core Libraries

- **Language:** Go 1.22+ (static binary, cross‑platform, excellent CLI and database support)
- **Database:** [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go SQLite, no CGO, easy distribution)
- **CLI Framework:** [cobra](https://github.com/spf13/cobra) (commands, help)
- **MCP SDK:** [https://github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) (or write a lightweight MCP server using the official Go library when available; the protocol is simple JSON‑RPC over stdio)
- **Editor Integration:** Use `$EDITOR` environment variable, fallback to `vim`/`nano`; or implement a simple interactive stdin loop for `park-day`.
- **Testing:** Standard `testing` package, table‑driven tests, integration tests that spin up an in‑memory SQLite DB.

### 6.2 Data Storage

- **Location:** `~/.towerctl/tower.db` (default, configurable via `config.yaml`).
- **Migrations:** Embedded SQL migration files (e.g., `001_initial.sql`, `002_add_tasks.sql`) applied at startup using a naive migration runner. Use a `schema_version` table.
- **Backups:** Exports (markdown/yaml) serve as manual backups. Automatic daily backup to `~/.towerctl/backups/` can be added later.

### 6.3 Project Repository Layout

```
towerctl/
├── cmd/
│   └── towerctl/
│       └── main.go
├── internal/
│   ├── db/
│   │   ├── migrations/
│   │   ├── sqlite.go
│   │   └── models.go
│   ├── commands/
│   │   ├── root.go
│   │   ├── project.go
│   │   ├── park.go
│   │   ├── morning.go
│   │   └── ...
│   ├── mcp/
│   │   ├── server.go
│   │   └── tools.go
│   └── export/
│       └── export.go
├── docs/
│   ├── hermes-agent/
│   │   ├── tools.yaml
│   │   ├── system-prompt.md
│   │   └── README.md
│   └── user-guide.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 7. MCP Integration & Documentation for Hermes‑Agent

### 7.1 Tool Definitions

A YAML/JSON file (`docs/hermes-agent/tools.yaml`) will contain a ready‑to‑use tool configuration that the Hermes‑agent can ingest directly. Example:

```yaml
tools:
  - name: get_active_project_context
    description: "Get structured active-project context for agent planning."
    command: "towerctl morning --format json"
    # OR if using MCP server:
    # mcp_tool: get_morning_brief
  - name: park_context
    description: "Save a context snapshot for a project. Optionally refresh priority context. Parameters: project_id, stopped_where, what_changed, next_action, blockers, files_links, need_help, priority, priority_reason, deadline, status."
    command: "towerctl park {{project_id}} --stopped '{{stopped_where}}' --next '{{next_action}}' --blocker '{{blockers}}' --changed '{{what_changed}}' --files '{{files_links}}' --help '{{need_help}}' --priority '{{priority}}' --priority-reason '{{priority_reason}}' --deadline '{{deadline}}' --status '{{status}}'"
  # ... more tools
```

For **MCP mode**, the agent config will point to the MCP server process:

```yaml
mcp_servers:
  - name: tower
    command: "towerctl serve-mcp"
```

We will also provide a **system prompt fragment** (`docs/hermes-agent/system-prompt.md`) that teaches the agent when and how to use these tools, e.g., “During planning, call `get_active_project_context`. When parking context, guide the user through the conversation yourself and store the result with `park_context` or `park_day_update`.”

### 7.2 Documentation for Agent Self‑Integration

The `README.md` in `docs/hermes-agent/` will explain how to set up Hermes‑agent with `towerctl`:

- Copy `tools.yaml` into the agent’s tool registry.
- Append the system prompt to the agent’s base instructions.
- Start the agent; it will automatically call the tools at the appropriate times.

This ensures the user can “feed him that directly, and he will integrate the tools itself” without manual coding.

---

## 8. Database Schema (Initial)

```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT DEFAULT 'personal' CHECK(type IN ('work','side_hustle','personal')),
    status TEXT DEFAULT 'active' CHECK(status IN ('active','blocked','paused','done')),
    priority INTEGER DEFAULT 3 CHECK(priority BETWEEN 1 AND 5),
    priority_reason TEXT DEFAULT '',
    last_prioritized_at TEXT,
    deadline TEXT,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stopped_where TEXT DEFAULT '',
    what_changed TEXT DEFAULT '',
    next_action TEXT DEFAULT '',
    blockers TEXT DEFAULT '',
    files_links TEXT DEFAULT '',
    need_help TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_snapshots_project ON snapshots(project_id, created_at DESC);
```

---

## 9. Non‑Functional Requirements

### 9.1 Reliability & Stability

- **Graceful error handling**: all database operations wrapped in transactions; no partial writes.
- **Migration safety**: never apply a migration twice; rollback on failure is not required (SQLite’s transactional DDL helps), but we will error loudly.
- **Concurrency**: SQLite in WAL mode allows reads during a write; since it’s a single‑user local tool, contention is minimal. We’ll use connection pooling (e.g., `sql.DB`) with `SetMaxOpenConns(1)` to avoid “database locked” errors.
- **Data integrity**: foreign keys enabled by default; `ON DELETE CASCADE` for dependent tables.

### 9.2 Performance

- Startup time < 50ms (no heavy computation).
- Morning brief query must complete in under 200ms even with hundreds of snapshots.

### 9.3 Usability

- Clear help text for each command (`--help`).
- Sensible defaults (e.g., `towerctl park` without arguments lists active projects to choose from).
- Configuration via `~/.towerctl/config.yaml` with defaults for database path, editor, export format, etc.

### 9.4 Security & Privacy

- All data stays local; no network calls unless explicitly adding an integration later (like fetching repo metadata). The MCP server uses stdio, not network sockets.
- No telemetry.

---

## 10. Development Phases

### Phase 1 – MVP (v0.1.0)

- [ ] Project add/list/show/update
- [ ] Park context with editor template
- [ ] `morning` and `next` commands that return structured project state for agent consumption
- [ ] `check` command (static question list)
- [ ] `park-day` interactive (stdin) flow
- [ ] Export to markdown
- [ ] Basic MCP server with `get_active_project_context`, `get_project_list`, `get_project_detail`, `park_context`
- [ ] Core documentation + Hermes‑agent integration docs

### Later (only after MVP is useful)

- [ ] More filtering/search if daily use proves it is needed
- [ ] Backup/export improvements
- [ ] Additional integrations only when a real workflow needs them

---

## 11. Success Metrics

- **Time to recall next action** after opening terminal: < 5 seconds (via `towerctl next`).
- **Compliance with daily rhythm** measured by consistent morning/evening usage.
- **Hermes‑agent integration** works out‑of‑the‑box with the provided documentation (zero additional glue code).
- **Zero “lost context” incidents** where the user can’t remember what they were doing on a project.

---

## 12. Open Questions

1. Should `towerctl` support multiple users (e.g., team mode)? **Decision:** No, single‑user focus for v1.
2. Do we need a `towerctl dashboard` TUI? **Decision:** Defer; the agent will be the primary interface.
3. Should we support plugins/extensions for external tools? **Decision:** No for MVP; add only after daily use exposes a concrete need.

---

## 13. Conclusion

`towerctl` is a pragmatic, reliable tool that bridges the gap between an AI assistant’s reasoning and the messy, ever‑changing reality of project work. Built in Go with native MCP support, it will become an indispensable part of the Hermes‑agent ecosystem and any personal knowledge workflow.

Next step: set up the repository, scaffold the Go project, implement the SQLite layer, and ship the MVP commands. The Hermes‑agent documentation will be written alongside so that on Day 1, the agent can already use the tool.