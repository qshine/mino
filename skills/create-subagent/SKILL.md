---
name: create-subagent
description: Write a project subagent definition in .agents/agents/<name>.md when the user wants a reusable, specialised helper role (for example a reviewer, a read-only explorer, or a test runner) that spawn_agent can delegate tasks to.
---

# Create a subagent definition

A subagent definition is one Markdown file in the project's `.agents/agents/` directory. It gives a role a name, tells the parent agent when to use it, may narrow the tools the role can use, and holds the role guidance the subagent receives with every task.

## Steps

1. Agree on the role with the user: what it does, when it should be used, and which tools it needs. Ask when this is unclear instead of guessing.
2. Choose a name: lowercase ASCII letters and digits with single hyphens, at most 64 bytes. The file name is the name plus `.md`.
3. If `.agents/agents/<name>.md` already exists, read it and ask before replacing it.
4. If `.agents/agents/` does not exist, create it with `mkdir -p .agents/agents` (write_file does not create directories).
5. Write the file with write_file in mode `create`.
6. Tell the user that the definition becomes available in a new session (`/new`). Each session freezes its catalog of subagent definitions when it starts, so this session cannot use it yet.

## File format

```markdown
---
name: code-reviewer
description: Reviews changed files for correctness bugs and reports each finding with its file path. Use after a code change, before committing.
tools:
  - read_file
  - skill_view
---

You review code for correctness. Read the files named in the task and any code
they depend on. Report only real defects, each with the file path, what goes
wrong, and the input that triggers it. Do not edit files. End with a short list
of findings, most severe first, or say that you found none.
```

- `name` (required): a YAML string equal to the file name without `.md`.
- `description` (required): one YAML string of at most 1,024 characters. The parent agent sees only the name, description, and tools, so say what the role does and when to use it.
- `tools` (optional): a list of 1 to 64 exact tool names, such as `bash`, `read_file`, `write_file`, `edit_file`, `skill_view`, `memory_view`, or an MCP tool's `mcp_<server>_<tool>` name. Omit it to give the role every tool a subagent normally has. The list only narrows: it never grants approval or adds a tool, and `spawn_agent` and `memory_save` cannot be listed. A listed tool that is not available when the subagent runs is left out.
- Body: the role guidance. Write it as instructions for the role: what to focus on, how to work, and what to return. The subagent cannot see the parent's conversation or ask the user questions, and only its final message comes back, so ask for a concise final report. The body is sent to the subagent as project-defined role text, below the user's instructions and tool approval.

Other front-matter keys, such as a model or a permission mode, are ignored: every subagent uses the session's model and permission mode. The whole file is limited to 64 KiB and its front matter to 16 KiB. Never put secrets in a definition.
