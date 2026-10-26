---
next:
  text: Setup and installation
  link: /getting-started
---

# Build an agent from scratch in Go

**Understand how an agent works through each interaction, then verify it with code and experiments.**

Mino is a project to build an agent from scratch in Go. This book follows it as each chapter adds one capability. Starting with a terminal conversation, you trace what the model receives and what the program handles, then check the result and its limits for yourself.

**Ready to read:** [Chapter 01: A terminal conversation](./chapters/01-terminal-chat.md) · [Chapter 02: JSONL conversation history](./chapters/02-jsonl-history.md) · [Chapter 03: The Agent loop and Bash](./chapters/03-tools-and-bash.md) · [Chapter 04: Recovering tool turns](./chapters/04-recovering-tool-turns.md) · [Chapter 05: Multiple sessions](./chapters/05-jsonl-sessions.md) · [Chapter 06: Context compaction](./chapters/06-context-compaction.md) · [Chapter 07: Skills](./chapters/07-skills.md) · [Chapter 08: File tools](./chapters/08-file-tools.md) · [Chapter 09: MCP tools](./chapters/09-mcp.md) · [Chapter 10: Guardrails](./chapters/10-guardrails.md) · [Chapter 11: Evals](./chapters/11-evals.md). All eleven chapters are released; the final chapter is available in `chapter-11` (version `0.11.0`).

## About the author

<div class="author-card">

**qqling | AI Builder** is a senior developer with time at large companies and startups, and a long-time AI practitioner focused on putting large models and agents to work. I want to build an agent of my own from scratch, and Mino is where that begins. For each new capability, the book explains the problem it solves, how it works, and how to verify the result, so readers new to agents can follow along one step at a time.

- GitHub: [qshine](https://github.com/qshine)
- WeChat official account: qqling的AI小院
- Xiaohongshu: [AI_qqling](https://www.xiaohongshu.com/user/profile/67039740000000001d031479)

<AuthorQr wechat="WeChat official account: qqling的AI小院" wechat-alt="QR code for the WeChat official account qqling的AI小院" xiaohongshu="Xiaohongshu: AI_qqling" xiaohongshu-alt="QR code for the Xiaohongshu account AI_qqling" />

</div>

## Who this book is for

This book is for readers new to agents who want to understand them by building. Reading the source requires Go variables, functions, and basic error handling; you can start with Chapter 01 even if model APIs are new to you. HTTP requests, context, and tool calls are introduced when the program needs them.

The exercises target macOS. Running a downloaded Mino release requires no Go installation; building or running Mino from source requires Go. See [setup and installation](./getting-started.md) for the steps.

## What each chapter gives you

1. **Observe an interaction.** Start with a concrete action, such as entering a question and seeing an answer, to establish the problem the chapter addresses.
2. **Separate the model's role from the program's.** Use diagrams and versioned source links to trace how input, context, and responses move.
3. **Verify with an experiment.** Inspect requests or run mock tests to establish where a capability comes from and when it can fail.

For example, Chapter 01 inspects requests to explain why two questions in the same terminal have no shared memory. Chapter 02 then verifies how saved history becomes part of the next request. A model's correct guess cannot replace a check of the program's behavior.

## What is ready to read

Chapters 01–10 were released on **2026-10-21** as `chapter-01` through `chapter-10`, with application versions `0.1.0` through `0.10.0`; `chapter-10` was republished on **2026-10-26**, and Chapter 11 followed on **2026-10-26** as `chapter-11` (version `0.11.0`). Chapter 10 consolidates the earlier controls into one guardrail decision with permission modes you choose; Chapter 11, the final chapter, measures the Agent with declared eval cases. Each released lesson identifies the source tag for its experiments.

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
| [09 MCP tools](./chapters/09-mcp.md) | How do tools hosted by remote services or local processes join the loop, and what does each approval decide? |
| [10 Guardrails](./chapters/10-guardrails.md) | How are actual permissions enforced as one policy in code? |
| [11 Evals](./chapters/11-evals.md) | How do you check repeatably whether a change to the SOUL, a tool description, or the loop changed what the Agent does? |

## Read alongside the code

Each released chapter identifies its applicable version and uses its matching `chapter-NN` source tag. The newest chapter in this version of the book, Chapter 11, follows [chapter-11](https://github.com/qshine/mino/tree/chapter-11); the [source setup](./getting-started.md#try-chapter-11-from-source) explains how to run the same snapshot locally.

Release tags follow `chapter-NN`; the application version is `0.<chapter>.<patch>`. A patch such as `chapter-05.1` produces version `0.5.1` and updates the relevant lesson without creating an extra chapter.

Users of earlier `0.1.0` or `0.2.0` builds should follow the [update instructions](./getting-started.md#check-the-version-and-update).

Start with [setup and installation](./getting-started.md), or go directly to [Chapter 01](./chapters/01-terminal-chat.md).
