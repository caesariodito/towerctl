# towerctl

`towerctl` is a local, single-user context management layer for Hermes-agent and other AI agents. It stores projects, snapshots, and priority context in a SQLite database, then exposes the state through a human CLI and MCP tools.

`towerctl` stays dumb: it stores and validates context. Hermes-agent owns reasoning, planning, prioritization, and conversation flow.

## Install from source

```bash
go build -o towerctl ./cmd/towerctl
```

Put the resulting `towerctl` binary on your `PATH`.

## Quick start

```bash
towerctl project add "Tower Control" --type side_hustle --priority 4 --priority-reason "Need agent context layer"
towerctl park tower-control --stopped "Defined MVP" --next "Wire Hermes-agent" --status active
towerctl morning --format json
```

## Core commands

```bash
towerctl project add <name> [--type work|side_hustle|personal] [--priority <1-5>] [--priority-reason <text>] [--deadline <date>]
towerctl project list [--status active|blocked|paused|done] [--format json]
towerctl project show <id> [--format json]
towerctl project update <id> [--status active|blocked|paused|done] [--priority <1-5>] [--priority-reason <text>] [--deadline <date>] [--notes <text>]

towerctl park <project-id> --stopped "..." --changed "..." --next "..." --blocker "..." --files "..." --help "..."
towerctl morning --format json
towerctl next --format json
towerctl check --format json
towerctl park-day
towerctl export --format markdown
```

## Hermes-agent integration

Hermes-agent setup lives in [`docs/hermes-agent/README.md`](docs/hermes-agent/README.md).

That guide covers:

- `docs/hermes-agent/tools.yaml`
- `docs/hermes-agent/system-prompt.md`
- running `towerctl serve-mcp`

## Storage

Data is local-first and stored at:

```text
~/.towerctl/tower.db
```

Exports are written to:

```text
~/.towerctl/exports/towerctl.md
```

## Development

```bash
go test ./...
```

## Design decision

See [`docs/adr/0001-keep-towerctl-as-dumb-context-store.md`](docs/adr/0001-keep-towerctl-as-dumb-context-store.md).
