# Keep towerctl as a dumb context store

`towerctl` stores, validates, and returns structured project context, but does not perform AI reasoning or depend on an LLM. Hermes-agent or another harness agent owns planning, prioritization, conversation flow, and interpretation, while `towerctl` remains the local source of truth accessed through CLI and MCP tools.
