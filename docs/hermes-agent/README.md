# Hermes-agent integration for towerctl

This guide wires Hermes-agent to `towerctl` as dumb local context storage.

## What you need

- `towerctl` binary on your `PATH`
- local access to `towerctl serve-mcp`
- `docs/hermes-agent/tools.yaml`
- `docs/hermes-agent/system-prompt.md`

## How to integrate

1. Start MCP mode:

```bash
towerctl serve-mcp
```

2. Add `docs/hermes-agent/tools.yaml` to Hermes-agent tool config.
3. Append `docs/hermes-agent/system-prompt.md` to Hermes-agent instructions.
4. Let Hermes-agent call tools during planning, checking, and parking.

## Included tools

- `get_active_project_context`
- `get_project_list`
- `get_project_detail`
- `park_context`
- `start_park_day`
- `park_day_update`

## Mental model

- Hermes-agent reasons.
- towerctl stores state.
- towerctl does not host AI or LLM logic.
- Hermes-agent should not invent facts; it should ask user and store updates.

## Files

- [`tools.yaml`](./tools.yaml)
- [`system-prompt.md`](./system-prompt.md)
