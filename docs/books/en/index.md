---
next:
  text: Setup and installation
  link: /getting-started
---

# Build an agent from scratch in Go

**Understand how an agent works through each interaction, then verify it with code and experiments.**

By [qqling | AI Builder](./about-author.md)

Mino is a project to build an agent from scratch in Go. This book follows it as each chapter adds one capability. Starting with a terminal conversation, you trace what the model receives and what the program handles, then check the result and its limits for yourself.

**Ready to read:** [Chapter 01: A terminal conversation](./chapters/01-terminal-chat.md) · [Chapter 02: JSONL conversation history](./chapters/02-jsonl-history.md) · [Chapter 03: The Agent loop and Bash](./chapters/03-tools-and-bash.md) · [Chapter 04: Recovering tool turns](./chapters/04-recovering-tool-turns.md) · [Chapter 05: Multiple sessions](./chapters/05-jsonl-sessions.md) · [Chapter 06: Context compaction](./chapters/06-context-compaction.md) · [Chapter 07: Skills](./chapters/07-skills.md) · [Chapter 08: File tools](./chapters/08-file-tools.md). This snapshot covers the book up to Chapter 08, release `chapter-08` (version `0.8.0`); later chapters arrive in their own releases.

## Who this book is for

This book is for readers new to agents who want to understand them by building. Reading the source requires Go variables, functions, and basic error handling; you can start with Chapter 01 even if model APIs are new to you. HTTP requests, context, and tool calls are introduced when the program needs them.

The exercises target macOS. Running a downloaded Mino release requires no Go installation; building or running Mino from source requires Go. See [setup and installation](./getting-started.md) for the steps.

## What each chapter gives you

1. **Observe an interaction.** Start with a concrete action, such as entering a question and seeing an answer, to establish the problem the chapter addresses.
2. **Separate the model's role from the program's.** Use diagrams and versioned source links to trace how input, context, and responses move.
3. **Verify with an experiment.** Inspect requests or run mock tests to establish where a capability comes from and when it can fail.

For example, Chapter 01 inspects requests to explain why two questions in the same terminal have no shared memory. Chapter 02 then verifies how saved history becomes part of the next request. A model's correct guess cannot replace a check of the program's behavior.

## What is ready to read

This snapshot is `chapter-08` (application version `0.8.0`), released on **2026-10-21**. Rows after Chapter 08 describe later releases and link to the [chapter roadmap](./plan-todo-chapters.md). Each released lesson identifies the source tag for its experiments.

| Stage | The question you will explore |
| --- | --- |
| [01 A terminal conversation](./chapters/01-terminal-chat.md) | How does a line of input become an HTTP request and then an answer? |
| [02 JSONL conversation history](./chapters/02-jsonl-history.md) | What survives a restart, and which saved records become model context? |
| [03 The Agent loop and Bash](./chapters/03-tools-and-bash.md) | Who approves and executes a command, and how does its result return to the model? |
| [04 Recovering tool turns](./chapters/04-recovering-tool-turns.md) | When a command may have run but its result was lost, how does the model find out? |
| [05 Multiple sessions](./chapters/05-jsonl-sessions.md) | How do you select one conversation's context, keep another, and safely clear the current one? |
| [06 Context compaction](./chapters/06-context-compaction.md) | How do manual and automatic compaction share a budget and preserve useful context? |
| [07 Skills](./chapters/07-skills.md) | How does a fixed catalog guide the model to request task knowledge with `skill_view`? |
| [08 File tools](./chapters/08-file-tools.md) | How can the program confine the model's project-file reads and bind each approved write to the file you saw? |
| [09 MCP tools](plan-todo-chapters.md) | How do tools hosted by remote services or local processes join the loop, and what does each approval decide? |
| [10 Guardrails](plan-todo-chapters.md) | How are actual permissions enforced as one policy in code? |

## Read alongside the code

Each released chapter identifies its applicable version and uses its matching `chapter-NN` source tag. The newest chapter in this version of the book, Chapter 08, follows [chapter-08](https://github.com/qshine/mino/tree/chapter-08); the [source setup](./getting-started.md#learn-from-the-source) explains how to run the same snapshot locally.

Release tags follow `chapter-NN`; the application version is `0.<chapter>.<patch>`. A patch such as `chapter-05.1` produces version `0.5.1` and updates the relevant lesson without creating an extra chapter.

Users of earlier `0.1.0` or `0.2.0` builds should follow the [update instructions](./getting-started.md#check-the-version-and-update).

Start with [setup and installation](./getting-started.md), or go directly to [Chapter 01](./chapters/01-terminal-chat.md).

To learn why I started Mino or follow future work, visit [about the author](./about-author.md).
