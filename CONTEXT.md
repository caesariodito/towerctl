# towerctl

`towerctl` is a dumb local operational-memory system for helping Hermes-agent manage a user's personal SOP workflow without bloating the user's mind or the assistant's long-term memory.

## Language

**Operational Memory**:
Short-lived project state that helps a user resume, prioritize, and park work across projects and personal targets. It is not permanent knowledge, documentation, or assistant long-term memory.
_Avoid_: Project memory, knowledge base, second brain

**SOP Workflow**:
The user's daily operating rhythm for choosing targets, checking progress, and parking context. Timing of each step is decided by Hermes-agent.
_Avoid_: Reminder system, task manager, productivity app

**Context Managing Layer**:
The local source of structured project state used by Hermes-agent or other AI agents. It stores and scores data but does not contain AI or LLM dependencies.
_Avoid_: AI assistant, agent brain, LLM app

**Project**:
A work, side-hustle, or personal workstream the user wants to track independently, with its own status, priority, priority reason, and latest resume point.
_Avoid_: Workspace, repo, initiative, context

**Personal Target**:
A non-project outcome the user wants Hermes-agent to keep visible during daily planning. Personal targets are outside MVP unless they need distinct behavior from projects.
_Avoid_: Goal, habit, reminder, context

**Snapshot**:
A point-in-time parking record describing where work stopped, what changed, next action, blockers, links, and help needed.
_Avoid_: Context, note, log entry
