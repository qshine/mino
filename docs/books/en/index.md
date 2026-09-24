---
next:
  text: Setup and installation
  link: /getting-started
---

# Build an agent from scratch in Go

**An illustrated book that grows alongside Mino, one runnable chapter at a time.**

How does a model API become a program you can talk to in a terminal? How does that program acquire memory, call tools, save sessions, and act within clear permissions? This book follows those questions, adding one capability at a time.

Begin with a short interaction, then follow what the user supplies, what the model receives, and what the program does next. Focused diagrams and source links explain how data and control move; small experiments make the boundaries observable.

## Who this book is for

You know Go variables, functions, and basic error handling, and want to understand how agents work. You can start with Chapter 01 even if model APIs are new to you. HTTP requests, context, and tool calls are introduced when the program needs them.

Mino targets macOS and uses the official OpenAI Go SDK for API communication. The book builds Mino's interaction flow and later Agent loop in Go. Reading the book requires no website tooling, and running a downloaded Mino release requires no Go installation. Go is needed to work with the application source; Node.js is needed only to maintain the book website.

## What each chapter gives you

Each lesson follows one concrete interaction through the minimum code needed to understand it, then gives you an experiment to check the result. It makes clear what works and what problem remains. Installation and configuration live in [getting started](./getting-started.md); chapters link there without repeating the setup walkthrough.

## What is ready to read

Chapters 01 and 02 are complete in the source and book. Chapter 01 follows the **v0.1.0 streaming reissue**: send the loaded identity and current question, display incoming answer fragments, and check completion. It explains why that version's independent requests have no memory. Users of earlier `0.1.0` builds should follow the [update instructions](./getting-started.md#check-the-version-and-update).

Chapter 02, available in **v0.2.0**, saves one conversation in a JSONL history file and restores it after restart. Records within each turn share a `turn_id`; only completed turns become context for the next question. Chapter 03 onward remains planned. The [chapter roadmap](./plan-todo-chapters.md) separates completed work from future capabilities.

| Stage | The question you will explore |
| --- | --- |
| [01 A terminal conversation](./chapters/01-terminal-chat.md) | How does a line of input become an HTTP request and then an answer? |
| [02 JSONL conversation history](./chapters/02-jsonl-history.md) | What survives a restart, and which saved records become model context? |
| 03–04 Tools and multiple sessions | Who executes tools? How do you start or restore a separate conversation? |
| 05–06 Context management | How can older interactions be summarized, and when should that happen? |
| 07–09 Extensions and guardrails | How do knowledge and tools connect, and how are actual permissions enforced? |

## Read alongside the code

Each chapter identifies its applicable version. Important source links point to a checked release tag or an immutable development commit, so the implementation behind a lesson remains available as the project evolves.

Versions follow `0.<chapter>.<patch>`: Chapter 01 starts at `0.1.0`, a fix becomes `0.1.1`, and Chapter 02 starts at `0.2.0`. Small fixes update the relevant text and diagrams without creating an extra chapter.

Start with [setup and installation](./getting-started.md), or go directly to [Chapter 01](./chapters/01-terminal-chat.md).
