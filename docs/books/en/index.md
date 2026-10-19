---
next:
  text: Setup and installation
  link: /getting-started
---

# Build an agent from scratch in Go

**Understand how an agent works through each interaction, then verify it with code and experiments.**

By [qqling | AI Builder](./about-author.md)

Mino is a project to build an agent from scratch in Go. This book follows it as each chapter adds one capability. Starting with a terminal conversation, you trace what the model receives and what the program handles, then check the result and its limits for yourself.

**Ready to read:** [Chapter 01: A terminal conversation](./chapters/01-terminal-chat.md) · [Chapter 02: JSONL conversation history](./chapters/02-jsonl-history.md) · [Chapter 03: Tools and Bash](./chapters/03-tools-and-bash.md) · [Chapter 04: Multiple sessions](./chapters/04-jsonl-sessions.md) · [Chapter 05: Context compaction](./chapters/05-context-compaction.md) · [Chapter 06: Skills](./chapters/06-skills.md) · [Chapter 07: File tools](./chapters/07-file-tools.md) · [Chapter 08: MCP tools](./chapters/08-mcp.md) · [Chapter 09: Guardrails](./chapters/09-guardrails.md) · [Chapter 10: Evals](./chapters/10-evals.md). All ten chapters are released; the latest is `chapter-10` (version `0.10.0`).

## Who this book is for

This book is for readers new to agents who want to understand them by building. Reading the source requires Go variables, functions, and basic error handling; you can start with Chapter 01 even if model APIs are new to you. HTTP requests, context, and tool calls are introduced when the program needs them.

The exercises target macOS. Running a downloaded Mino release requires no Go installation; building or running Mino from source requires Go. See [setup and installation](./getting-started.md) for the steps.

## What each chapter gives you

1. **Observe an interaction.** Start with a concrete action, such as entering a question and seeing an answer, to establish the problem the chapter addresses.
2. **Separate the model's role from the program's.** Use diagrams and versioned source links to trace how input, context, and responses move.
3. **Verify with an experiment.** Inspect requests or run mock tests to establish where a capability comes from and when it can fail.

For example, Chapter 01 inspects requests to explain why two questions in the same terminal have no shared memory. Chapter 02 then verifies how saved history becomes part of the next request. A model's correct guess cannot replace a check of the program's behavior.

## What is ready to read

Chapters 01–05 were released on **2026-09-26** as `chapter-01` through `chapter-05`, with application versions `0.1.0` through `0.5.0`. Chapter 05 adds manual and automatic context compaction. Chapter 06 followed on **2026-10-11** as `chapter-06` (version `0.6.0`), Chapter 07 on **2026-10-11** as `chapter-07` (version `0.7.0`), Chapter 08 on **2026-10-11** as `chapter-08` (version `0.8.0`), and Chapter 09 on **2026-10-12** as `chapter-09` (version `0.9.0`). Chapter 10, [Measure the Agent with evals](./chapters/10-evals.md), followed on **2026-10-19** as `chapter-10` (version `0.10.0`). Each released lesson identifies the source tag for its experiments.

| Stage | The question you will explore |
| --- | --- |
| [01 A terminal conversation](./chapters/01-terminal-chat.md) | How does a line of input become an HTTP request and then an answer? |
| [02 JSONL conversation history](./chapters/02-jsonl-history.md) | What survives a restart, and which saved records become model context? |
| [03 Tools and Bash](./chapters/03-tools-and-bash.md) | Who approves and executes a command, and what happens if its result is lost? |
| [04 Multiple sessions](./chapters/04-jsonl-sessions.md)  | How do you select one conversation's context, keep another, and safely clear the current one? |
| [05 Context compaction](./chapters/05-context-compaction.md) | How do manual and automatic compaction share a budget and preserve useful context? |
| [06 Skills](./chapters/06-skills.md) | How does a fixed catalog guide the model to request task knowledge with `skill_view`? |
| [07 File tools](./chapters/07-file-tools.md) | How can the program confine the model's project-file reads and bind each approved write to the file you saw? |
| [08 MCP tools](./chapters/08-mcp.md) | How do tools hosted by remote services or local processes join the loop, and what does each approval decide? |
| [09 Guardrails](./chapters/09-guardrails.md) | How are actual permissions enforced as one policy in code? |
| [10 Evals](./chapters/10-evals.md) | How do you repeat an interaction and grade what the Agent did, against a scripted model or a live one? |

## Read alongside the code

Each released chapter identifies its applicable version and uses its matching `chapter-NN` source tag. The latest release, Chapter 10, follows [chapter-10](https://github.com/qshine/mino/tree/chapter-10); its [source setup](./getting-started.md#try-chapter-10-from-source) explains how to run the same snapshot locally.

Release tags follow `chapter-NN`; the application version is `0.<chapter>.<patch>`. A patch such as `chapter-04.1` produces version `0.4.1` and updates the relevant lesson without creating an extra chapter.

Users of earlier `0.1.0` or `0.2.0` builds should follow the [update instructions](./getting-started.md#check-the-version-and-update).

Start with [setup and installation](./getting-started.md), or go directly to [Chapter 01](./chapters/01-terminal-chat.md).

To learn why I started Mino or follow future work, visit [about the author](./about-author.md).
