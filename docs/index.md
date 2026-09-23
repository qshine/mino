---
next:
  text: Setup and installation
  link: /getting-started
---

# Build an agent from scratch in Go

**An illustrated book that grows alongside Mino, one runnable chapter at a time.**

How does a model API become a program you can talk to in a terminal? How does that program acquire memory, call tools, save sessions, and act within clear permissions? This book follows those questions, adding one capability at a time.

Run each chapter's result first. Then follow its diagrams and code to understand why it works. The diagrams explain how data and control move through the program; terminal examples show behavior you can observe yourself.

## Who this book is for

You know Go variables, functions, and basic error handling, and want to understand how agents work. You can start with Chapter 01 even if model APIs are new to you. HTTP requests, context, and tool calls are introduced when the program needs them.

Mino targets macOS, and the application currently uses only Go's standard library. Reading the book requires no website tooling, and running a downloaded Mino release requires no Go installation. Go is needed to work with the application source; Node.js is needed only to maintain the book website.

## What each chapter gives you

1. **A concrete problem:** Where does the previous chapter's program fall short?
2. **An outcome:** What new behavior will the program have by the end?
3. **An explanation with diagrams:** A concept map, flowchart, or sequence diagram follows an operation through the system.
4. **An implementation path:** Find the responsible code and understand its tradeoffs.
5. **Experiments and checks:** Observe the result and use tests to verify its boundaries.
6. **What this chapter completed:** Separate working capabilities from the next chapter's open questions.

## What is ready to read

Chapter 01 is complete. It applies to **0.1.x** and has been checked against **v0.1.1**. It covers independent terminal questions, first-run configuration, and installation and updates.

Chapter 02 and later chapters are planned, not implemented. The [chapter roadmap](./plan-todo-chapters.md) tracks progress; future capabilities are never presented as features that already work.

| Stage | The question you will explore |
| --- | --- |
| [01 A terminal conversation](./chapters/01-terminal-chat.md) | How does a line of input become an HTTP request and then an answer? |
| 02–04 Context, tools, and sessions | Who remembers the conversation? Who executes tools? What survives a restart? |
| 05–06 Context management | How can older interactions be summarized, and when should that happen? |
| 07–09 Extensions and guardrails | How do knowledge and tools connect, and how are actual permissions enforced? |

## Read alongside the code

Each chapter identifies its applicable version. Important source links point to a checked release tag, so the implementation behind a lesson remains available as the project evolves.

Versions follow `0.<chapter>.<patch>`: Chapter 01 starts at `0.1.0`, a fix becomes `0.1.1`, and Chapter 02 starts at `0.2.0`. Small fixes update the relevant text and diagrams without creating an extra chapter.

Start with [setup and installation](./getting-started.md), or go directly to [Chapter 01](./chapters/01-terminal-chat.md).
