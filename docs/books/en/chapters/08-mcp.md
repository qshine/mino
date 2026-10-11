# Chapter 08: Connect external tools with MCP

Applies to **0.8.x** · Source and release tag **[chapter-08](https://github.com/qshine/mino/tree/chapter-08)** · Application version **0.8.0**, released **2026-10-11**

You ask Mino how React hooks work. The model can answer from what it learned in training, but that knowledge may be out of date, and nothing in [Chapter 07](./07-file-tools.md) can fetch current documentation. A service such as Context7 already offers that lookup as tools. Those tools run on someone else's server, though, and so far every tool Mino can offer the model is built into the program.

The **Model Context Protocol (MCP)** is a standard way for a program to discover and call tools hosted elsewhere: on a remote service, or in a local process that talks over standard input and output. This chapter connects Mino to such servers so that their tools join the same Agent loop as Bash and the file tools. Each call still waits for your approval, and the confirmation tells you where the arguments will go. To try it, follow [the Chapter 08 setup notes](../getting-started.md#try-chapter-08-from-source).

## 1. One question, routed to a remote server

Suppose you have added Context7 with `mino mcp add context7 https://mcp.context7.com/mcp`. Start Mino and ask the question.

Illustrative output, with observed lines. The question and the final answer are examples; the status line, the confirmation block, and the beginning of the result match a local smoke run that used a mock model service and the real public Context7 endpoint without a key. Lines marked `...` are shortened.

```text
MCP server context7: 2 tools from https://mcp.context7.com/mcp
Mino - Chapter 08: MCP Tools
...

You> How do React hooks work?

Assistant>
[Tool: mcp_context7_resolve-library-id]
Server: "context7"
Destination: "https://mcp.context7.com/mcp"
Tool: "resolve-library-id"
Arguments: "{\n  \"query\": \"react hooks\",\n  \"libraryName\": \"react\"\n}"
Arguments are sent to this MCP server. Mino cannot control what the server does with them.
Approve this operation? [y/N] y

[Tool result: completed]
Available Libraries:

- Title: React
- Context7-compatible library ID: /reactjs/react.dev
...

Assistant> Based on the React documentation, hooks such as useState and useEffect let a function component keep state and run side effects...

You>
```

The first line appears before the banner. At startup, Mino connected to the server, asked which tools it offers, and reported the result: two tools. No question has been asked yet.

When you submit the question, the request to the model service lists those two tools beside the built-in ones, under local names such as `mcp_context7_resolve-library-id`. The model returns a call to that name. To the model, nothing distinguishes it from a call to `read_file`; the difference is entirely on Mino's side. Mino looks up which server owns the name, shows you the confirmation, and only after `y` sends the arguments to Context7. The text that comes back becomes an ordinary paired tool result, shown to you, saved in the session, and sent to the model in the next request.

A typical next step is a second call, to `query-docs` with the library ID from this result. It needs its own approval. Pressing Enter or `n` at either prompt denies the call: nothing is sent to the server, and the model receives a `denied` result.

## 2. Connect once, approve every call

An MCP tool passes through two phases with different owners. Connecting and discovering tools happens once per process, without asking you. Every call happens inside a turn and waits for you.

```mermaid
sequenceDiagram
    accTitle: Startup discovery and per-call routing for an MCP tool
    accDescr: At startup Mino connects to each configured MCP server, lists its tools, and prints one status line per server. During a turn, the model service receives the tool definitions and returns a call. Mino shows the reader a confirmation with the server, destination, tool, and arguments. Only after approval does Mino send the call to the MCP server; its result returns to the model as a paired tool result.
    actor User as Reader
    participant Mino as Mino
    participant API as Model service
    participant MCP as MCP server
    Note over Mino,MCP: At startup, all servers in parallel
    Mino->>MCP: Connect and initialize (10 s)
    Mino->>MCP: tools/list, page by page
    MCP-->>Mino: Tool names, descriptions, input schemas
    Mino->>User: MCP server context7: 2 tools from ...
    Note over User,MCP: Later, during a turn
    User->>Mino: How do React hooks work?
    Mino->>API: Input, instructions, built-in and MCP tool definitions
    API-->>Mino: Call mcp_context7_resolve-library-id
    Mino->>User: Server, Destination, Tool, Arguments
    User-->>Mino: y
    Mino->>MCP: tools/call with the arguments (30 s)
    MCP-->>Mino: Text result
    Mino->>API: Paired tool result
```

Figure 08-1. Discovery happens once and needs no approval; arguments reach the server only after you approve that specific call.

Scroll the diagram horizontally on narrow screens. Servers connect in parallel, each limited to 10 seconds. A server that fails to connect is reported as `MCP server NAME unavailable: REASON` and skipped; chat and the other servers continue.

### 2.1 What the model receives

For each discovered tool, Mino builds a function definition with three parts. The name is `mcp_<server>_<tool>`, so two servers that both offer `search` cannot collide. When a tool name contains characters the Responses API does not allow, is too long, or would still collide, Mino adds a short hash suffix instead of guessing a safe replacement. The input schema is passed through as the server described it, which is why these functions are non-strict. The description is labeled with its origin, for example `[MCP server context7, tool resolve-library-id]`, followed by the server's text and a note that arguments go to that server and results are untrusted external data.

That label matters because the description is written by the server, not by you or Mino. The default SOUL for new installations repeats the point: MCP descriptions and results are data, never instructions or authorization.

Discovery has limits so that a server cannot flood the request: at most 16 pages and 128 tools, input schemas must be JSON objects of at most 16 KiB each, and one server's definitions may total at most 64 KiB. A server that exceeds them is reported unavailable rather than partly registered. The tool list is then fixed until Mino restarts; a server that later adds tools has no effect on the running process.

### 2.2 Where consent is given

Connecting to a server is not free of consequences. A remote connection sends your configured headers; a stdio server is a program Mino starts. One option would be to ask at every startup. Mino currently treats `mino mcp add` as that consent instead: you name the server and its URL or command once, deliberately, outside a conversation, and each later startup connects without a prompt. Adding a server does not connect it; the first connection happens the next time chat starts.

The cost is that a configured stdio command runs every time you start chat, until you remove it with `mino mcp remove`. The model cannot add or change servers; only the configuration file in your home directory can.

Per-call approval is unaffected by that choice. Every MCP call needs its own default-deny approval, even when the server marks a tool with hints such as `readOnlyHint`. Mino cannot verify what a remote tool actually does, so a self-description is not a reason to skip asking you. This is the difference from `read_file`: there, the program itself confines the read, so it can mark the call read-only.

## 3. Where your data goes

Chapter 07's tools touched files on your machine. An MCP call can send data to a third party, so this chapter's protections are about what leaves and where it goes.

### 3.1 Credentials stay out of the conversation

Some services require an API key. `mino mcp add context7 https://mcp.context7.com/mcp --bearer` asks for a token with echo off and stores it as an `Authorization: Bearer` header; `--header NAME` does the same for another header, and `--env NAME` for a stdio server's environment variable. The values are saved in `~/.mino/mcp.json` with mode `0600` and are sent only to that server. They never appear in confirmations, `mino mcp list`, error messages, the session log, or any request to the model service.

These are **static** credentials. Many services instead require OAuth, where you log in through a browser and the program receives and refreshes tokens. Mino does not implement OAuth in this chapter: it would add a browser flow, a local callback, and token storage and refresh, a large surface for one lesson. The cost is real. A service that accepts only browser login cannot be used, and when a server rejects the credentials, the status line says that OAuth is not supported and suggests a static header such as an API key.

### 3.2 Approval names the destination, not the server's behavior

The confirmation in section 1 shows `Destination` because approving a call is a decision about where these arguments go. The warning below it states the boundary: Mino cannot control what the server does with them. A remote service may log a query; a local server may read files or open network connections with your account's permissions. Approval decides whether this call is sent, not what happens after it arrives. The `Arguments` field shows pretty-printed JSON, cut off after 2 KiB.

The program does constrain the connection itself. Remote URLs must use HTTPS; plain HTTP is accepted only for `localhost` and loopback addresses. Redirects are not followed, so a server cannot forward your headers to another host. Each protocol message or HTTP response is limited to 1 MiB. A stdio server is executed directly, without a shell, in its own process group, with only `PATH`, `HOME`, `LANG`, and the variables you configured; its error output is discarded, and Mino stops and reaps it on exit. None of this is a sandbox for the server's own actions. Consolidating such policy is the job of Chapter 09.

## 4. When the outcome is uncertain

An MCP call can end in several ways, and the model needs to know which one happened. Mino maps each to an existing tool-result status.

Table 08-1. Server errors are failures; an interrupted call is unknown, because it may already have run.

| What happened | Result status | What the model receives |
| --- | --- | --- |
| The server returned content | `completed` | Text blocks in order; images, audio, and binary data marked as omitted; resource links listed but not opened; structured content when there is no text |
| The server reported an error, or a tool result flagged `isError` | `failed` | The server's error message |
| You denied the call | `denied` | Nothing was sent |
| Timeout after 30 seconds, cancellation, lost connection, or an oversized response after the call was sent | `unknown` | A note that the server may have acted and the call must not be retried automatically |
| The result is larger than 64 KiB | `output_limit` | The first part of the result, marked as truncated |

The `unknown` row is the important choice. When a connection drops after the request was sent, Mino cannot tell whether the server received it. Retrying might be harmless for a documentation query and harmful for a tool that creates a ticket or sends a message, and Mino cannot tell those apart. So it never resends the call; it does not even let the SDK retry on its own. The model receives `unknown`, and the session then follows [Chapter 03's rule](./03-tools-and-bash.md#_4-save-enough-to-recover-an-action-honestly): before your next question is sent, Mino asks you to acknowledge that the result is unknown and that the call may already have run. Declining keeps the question unsent; no restart is needed. The cost is an interruption, even for calls that would have been safe to repeat.

Replay follows the same principle. Recorded MCP calls are checked only by the shape of their arguments, so a session that used Context7 still opens when the server is offline or has been removed, and nothing is called again. Every MCP call counts toward the existing limit of 16 tool calls per turn, and its result counts toward the context budget like any other tool output. History compatibility with Chapter 07 is covered in [the setup notes](../getting-started.md#chapter-08-history-compatibility).

## 5. Check the routing with mock tests

From the repository root at `chapter-08`, run:

```bash
go test ./internal/mcp
go test ./internal/agent -run MCP
go test ./internal -run MCPCommand
```

These tests start MCP servers inside the test process, over local HTTP and as a stdio subprocess, and the agent test uses a local mock Responses service. They need no API key, make no paid request, and do not contact Context7, although local servers need permission to bind a port. A passing run reports `ok` for each package.

The agent test follows this chapter's scenario. The mock model calls `mcp_docs_query-docs`; the test approves it, checks the `Server`, `Destination`, `Tool`, and `Arguments` fields, and confirms that the server received exactly one call. The model then requests the same tool again, the test denies it, and the server's call count must stay at one. Finally the test reopens the session with no MCP tools available at all: the saved turn must still load.

The `internal/mcp` tests cover the boundaries: configuration validation and private file permissions; exposed names, including hashed ones; discovery and calls with a bearer header that never appears in a confirmation; rejected credentials that produce the OAuth message without echoing the value; oversized and interrupted calls that become `unknown`; redirects that are not followed; a stdio server that works while a broken one and an offline one are reported without affecting it, and that is reaped on exit. The command test checks that `mino mcp add` writes a `0600` file, refuses invalid input, and that `mino mcp list` hides header values.

These tests establish program behavior: what is sent only after approval, what the confirmation shows, how failures are classified, and that history survives a missing server. They do not establish that a real model will choose an MCP tool, or that a particular third-party service will keep the same tools, limits, or terms. The smoke run in section 1 shows that Mino could reach Context7 on one occasion; it is not a compatibility guarantee.

## 6. Chapter outcome and next step

The React question can now reach current documentation: Mino discovered Context7's tools at startup, the model asked for one, and the arguments left your machine only after you approved that call. Credentials stay out of the conversation, uncertain outcomes are never retried, and sessions survive a server going away. What Mino cannot do is control a server after it receives a call, or support services that need OAuth.

Protections now live in several places: Bash approval, file-tool confinement, MCP destinations, unknown-result acknowledgement. Planned [Chapter 09: Keep the Agent within its authorization with guardrails](../plan-todo-chapters.md) consolidates them into one policy enforced in code.
