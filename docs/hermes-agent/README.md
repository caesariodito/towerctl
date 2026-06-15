# Hermes-agent integration for towerctl

This guide wires Hermes-agent to `towerctl` as dumb local context storage.

## What you need

- `towerctl` binary on your `PATH`
- local access to `towerctl serve-mcp`
- `docs/hermes-agent/tools.yaml`
- `docs/hermes-agent/system-prompt.md`

## Install towerctl

From repo root:

```bash
make test
make build
sudo cp bin/towerctl /usr/local/bin/towerctl
```

Verify it is on `PATH`:

```bash
towerctl --help
```

## How to enable MCP in Hermes-agent

Do not run `towerctl serve-mcp` manually in a separate terminal for normal agent use. MCP over stdio means Hermes-agent starts the process and talks to it over stdin/stdout.

Add this server config to Hermes-agent:

```yaml
mcp_servers:
  - name: tower
    command: "towerctl serve-mcp"
```

If `towerctl` is not on `PATH`, use an absolute path:

```yaml
mcp_servers:
  - name: tower
    command: "/absolute/path/to/towerctl serve-mcp"
```

Then append `docs/hermes-agent/system-prompt.md` to Hermes-agent instructions and expose the tools described in `docs/hermes-agent/tools.yaml`.

After Hermes-agent restarts/reloads config, ask it to call `get_active_project_context` to verify the connection.

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
