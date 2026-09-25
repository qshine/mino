---
next:
  text: Setup and installation
  link: /getting-started
---

# Build an agent from scratch in Go

**Understand how an agent works through each interaction, then verify it with code and experiments.**

By [qqling | AI Builder](./about-author.md)

Mino is a project to build an agent from scratch in Go. This book follows it as each chapter adds one capability. Starting with a terminal conversation, you trace what the model receives and what the program handles, then check the result and its limits for yourself.

**Ready to read:** [Chapter 01: A terminal conversation](./chapters/01-terminal-chat.md) · [Chapter 02: JSONL conversation history](./chapters/02-jsonl-history.md). Tools and multiple sessions are available in later chapter releases; see the [roadmap](./plan-todo-chapters.md).

## Who this book is for

This book is for readers new to agents who want to understand them by building. Reading the source requires Go variables, functions, and basic error handling; you can start with Chapter 01 even if model APIs are new to you. HTTP requests, context, and tool calls are introduced when the program needs them.

The exercises target macOS. Running a downloaded Mino release requires no Go installation; building or running Mino from source requires Go. See [setup and installation](./getting-started.md) for the steps.

## What each chapter gives you

1. **Observe an interaction.** Start with a concrete action, such as entering a question and seeing an answer, to establish the problem the chapter addresses.
2. **Separate the model's role from the program's.** Use diagrams and versioned source links to trace how input, context, and responses move.
3. **Verify with an experiment.** Inspect requests or run mock tests to establish where a capability comes from and when it can fail.

For example, Chapter 01 inspects requests to explain why two questions in the same terminal have no shared memory. Chapter 02 then verifies how saved history becomes part of the next request. A model's correct guess cannot replace a check of the program's behavior.

## What is ready to read

This edition contains Chapters 01–02, with matching `chapter-NN` release tags and numeric application versions. Use the source tag named in each lesson.

| Stage | The question you will explore |
| --- | --- |
| [01 A terminal conversation](./chapters/01-terminal-chat.md) | How does a line of input become an HTTP request and then an answer? |
| [02 JSONL conversation history](./chapters/02-jsonl-history.md) | What survives a restart, and which saved records become model context? |
| 03–04 Tools and multiple sessions | Who executes tools? How do you start or restore a separate conversation? |
| 05–06 Context management | How can older interactions be summarized, and when should that happen? |
| 07–09 Extensions and guardrails | How do knowledge and tools connect, and how are actual permissions enforced? |

## Read alongside the code

Each chapter identifies its applicable version. Important source links point to the matching `chapter-NN` release tag, so the implementation behind a lesson remains available as the project evolves.

Release tags follow `chapter-NN`; the application version is `0.<chapter>.<patch>`. A patch such as `chapter-04.1` produces version `0.4.1` and updates the relevant lesson without creating an extra chapter.

Users of earlier `0.1.0` builds should follow the [update instructions](./getting-started.md#check-the-version-and-update).

Start with [setup and installation](./getting-started.md), or go directly to [Chapter 01](./chapters/01-terminal-chat.md).

To learn why I started Mino or follow future work, visit [about the author](./about-author.md).
