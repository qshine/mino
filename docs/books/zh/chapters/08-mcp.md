# 第 08 章：用 MCP 接入外部工具

适用版本：**0.8.x** · 源码与发布标签：**[chapter-08](https://github.com/qshine/mino/tree/chapter-08)** · 程序版本：**0.8.0**（2026-10-11 发布）

你问 Mino：React 的 hooks 是怎么工作的？模型可以凭训练时学到的知识回答，但这些知识可能已经过时，而[第七章](./07-file-tools.md)里没有任何工具能取回最新文档。Context7 这样的服务已经把文档查询做成了工具。问题是，这些工具运行在别人的服务器上，而到目前为止，Mino 能提供给模型的工具全都内置在程序里。

**模型上下文协议（Model Context Protocol，MCP）**是一套标准方式，让程序发现并调用托管在别处的工具：可以在远端服务上，也可以在通过标准输入输出通信的本地进程里。本章让 Mino 连接这类服务，把它们的工具接入与 Bash、文件工具相同的 Agent 循环。每次调用仍要等你批准，确认界面会告诉你参数将发往何处。若要体验，请先阅读[第八章准备说明](../getting-started.md#从源码体验第八章)。

## 1. 一个问题，被路由到远端服务

假设你已经用 `mino mcp add context7 https://mcp.context7.com/mcp` 添加了 Context7。启动 Mino，提出问题。

交互示意，含实测输出行。提问和最后的回答是示例；状态行、确认界面和结果开头，与一次本地冒烟运行一致：那次运行使用模拟模型服务，并在不带密钥的情况下访问了真实的公开 Context7 端点。标为 `...` 的行做了省略。

```text
MCP server context7: 2 tools from https://mcp.context7.com/mcp
Mino - Chapter 08: MCP Tools
...

You> React 的 hooks 是怎么工作的？

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

Assistant> 根据 React 文档，useState、useEffect 等 hooks 让函数组件能够保存状态、执行副作用……

You>
```

第一行出现在横幅之前。启动时，Mino 连接了服务，询问它提供哪些工具，并报告结果：两个工具。这时你还没有提问。

你提交问题后，发给模型服务的请求会在内置工具旁边列出这两个工具，使用 `mcp_context7_resolve-library-id` 这样的本地名称。模型返回对这个名称的调用。在模型看来，它和调用 `read_file` 没有区别；不同之处全在 Mino 这一侧。Mino 查出这个名称属于哪个服务，向你展示确认界面，你输入 `y` 之后才把参数发给 Context7。返回的文本成为普通的配对工具结果：显示给你、保存到会话，并在下一次请求中交给模型。

接下来通常还有第二次调用：用这次结果里的库 ID 调用 `query-docs`。它需要单独批准。在任一提示处直接回车或输入 `n` 都表示拒绝：不向服务发送任何内容，模型收到 `denied` 结果。

## 2. 连接一次，每次调用都批准

一个 MCP 工具要经过两个阶段，由不同的人做主。连接和发现工具每个进程只发生一次，不会询问你。每次调用都发生在一轮问答之中，并等待你的决定。

```mermaid
sequenceDiagram
    accTitle: MCP 工具的启动发现与逐次调用路由
    accDescr: 启动时，Mino 连接每个已配置的 MCP 服务，列出其工具，并为每个服务打印一行状态。一轮问答中，模型服务收到工具定义并返回调用。Mino 向读者展示包含服务、目的地、工具和参数的确认界面。只有获批后，Mino 才把调用发给 MCP 服务；其结果作为配对工具结果交回模型。
    actor User as 读者
    participant Mino as Mino
    participant API as 模型服务
    participant MCP as MCP 服务
    Note over Mino,MCP: 启动时，所有服务并行
    Mino->>MCP: 连接并初始化（10 秒）
    Mino->>MCP: tools/list，逐页读取
    MCP-->>Mino: 工具名称、描述、输入 schema
    Mino->>User: MCP server context7: 2 tools from ...
    Note over User,MCP: 之后，在一轮问答中
    User->>Mino: React 的 hooks 是怎么工作的？
    Mino->>API: 输入、指令、内置与 MCP 工具定义
    API-->>Mino: 调用 mcp_context7_resolve-library-id
    Mino->>User: Server、Destination、Tool、Arguments
    User-->>Mino: y
    Mino->>MCP: 带参数的 tools/call（30 秒）
    MCP-->>Mino: 文本结果
    Mino->>API: 配对工具结果
```

图 08-1：发现只发生一次，无需批准；只有你批准这一次具体调用后，参数才会到达服务。

在窄屏上可以横向滚动图表。各服务并行连接，每个最多 10 秒。连接失败的服务会报告为 `MCP server NAME unavailable: REASON` 并被跳过；聊天和其他服务照常继续。

### 2.1 模型收到了什么

对每个发现的工具，Mino 生成一个由三部分组成的函数定义。名称是 `mcp_<server>_<tool>`，所以两个服务都提供 `search` 也不会冲突。如果工具名包含 Responses API 不允许的字符、过长，或者仍然冲突，Mino 会加上一个简短的哈希后缀，而不是去猜一个安全的替代名。输入 schema 按服务的描述原样传递，因此这些函数不使用严格模式。描述会标明来源，例如 `[MCP server context7, tool resolve-library-id]`，后面是服务提供的文字，再附上一句说明：参数会发给该服务，其结果是不可信的外部数据。

这个标记很重要，因为描述是服务写的，不是你或 Mino 写的。新安装使用的默认 SOUL 也强调了这一点：MCP 的描述和结果都是数据，绝不是指令或授权。

发现过程设有上限，防止某个服务塞满请求：最多 16 页、128 个工具；输入 schema 必须是 JSON 对象，每个最多 16 KiB；一个服务的全部定义合计最多 64 KiB。超出上限的服务会被报告为不可用，而不是只注册一部分。之后工具列表固定，直到 Mino 重启；服务后来新增的工具不会影响正在运行的进程。

### 2.2 同意在哪里给出

连接服务并非没有后果。远端连接会发送你配置的请求头；stdio 服务则是一个由 Mino 启动的程序。一种做法是每次启动都询问。Mino 目前的选择是把 `mino mcp add` 当作这份同意：你在对话之外，有意识地一次性写明服务及其 URL 或命令，之后每次启动都直接连接，不再提示。添加服务本身不会连接；第一次连接发生在下一次启动聊天时。

代价是：配置好的 stdio 命令会在每次启动聊天时运行，直到你用 `mino mcp remove` 删除它。模型无法添加或修改服务，只有你用户目录下的配置文件可以。

这个选择不影响逐次批准。每次 MCP 调用都需要单独的默认拒绝批准，即使服务给工具标注了 `readOnlyHint` 之类的提示。Mino 无法验证远端工具实际做了什么，因此服务的自我描述不能成为跳过询问的理由。这正是它和 `read_file` 的区别：后者的读取范围由程序自己限定，所以程序可以把调用标记为只读。

## 3. 你的数据去了哪里

第七章的工具只碰你机器上的文件。一次 MCP 调用可能把数据发给第三方，所以本章的保护关注的是：什么离开了本机，去了哪里。

### 3.1 凭证不进入对话

有些服务需要 API 密钥。`mino mcp add context7 https://mcp.context7.com/mcp --bearer` 会在关闭回显的情况下要求输入令牌，并保存为 `Authorization: Bearer` 请求头；`--header NAME` 对其他请求头做同样的事，`--env NAME` 则用于 stdio 服务的环境变量。这些值以 `0600` 权限保存在 `~/.mino/mcp.json`，只发给对应的服务。它们从不出现在确认界面、`mino mcp list`、错误信息、会话日志，或任何发给模型服务的请求中。

这些都是**静态**凭证。很多服务改用 OAuth：你在浏览器里登录，程序获取并刷新令牌。本章的 Mino 没有实现 OAuth：它需要浏览器流程、本地回调，以及令牌的存储与刷新，对一章来说面太大。代价是真实存在的：只接受浏览器登录的服务无法使用。服务拒绝凭证时，状态行会说明不支持 OAuth，并建议改用 API 密钥这类静态请求头。

### 3.2 批准说明目的地，而不约束服务的行为

第 1 节的确认界面显示 `Destination`，因为批准一次调用，就是决定这些参数发往哪里。下面那句警告写明了边界：Mino 无法控制服务拿这些参数做什么。远端服务可能记录查询内容；本地服务可能以你的账户权限读取文件或建立网络连接。批准决定的是这次调用发不发，而不是它到达之后会发生什么。`Arguments` 字段显示格式化后的 JSON，超过 2 KiB 的部分会被截掉。

程序确实约束了连接本身。远端 URL 必须使用 HTTPS，只有 `localhost` 和回环地址允许明文 HTTP。不跟随重定向，因此服务无法把你的请求头转给另一台主机。每条协议消息或 HTTP 响应最多 1 MiB。stdio 服务直接执行、不经过 shell，运行在自己的进程组中，只带 `PATH`、`HOME`、`LANG` 和你配置的变量；它的错误输出被丢弃，Mino 退出时会停止并回收它。这些都不是约束服务自身行为的沙箱。统一这类策略，是第九章的任务。

## 4. 结果不确定时

一次 MCP 调用有多种结束方式，模型需要知道实际是哪一种。Mino 把每种情况映射到已有的工具结果状态。

表 08-1：服务报错算失败；中断的调用算未知，因为它可能已经执行。

| 发生了什么 | 结果状态 | 模型收到什么 |
| --- | --- | --- |
| 服务返回了内容 | `completed` | 按顺序排列的文本块；图片、音频和二进制数据标记为已省略；资源链接只列出、不打开；没有文本时使用结构化内容 |
| 服务报告错误，或工具结果带有 `isError` 标记 | `failed` | 服务给出的错误信息 |
| 你拒绝了调用 | `denied` | 没有发送任何内容 |
| 调用发出后 30 秒超时、被取消、连接断开，或响应超限 | `unknown` | 一段说明：服务可能已经执行，不得自动重试 |
| 结果超过 64 KiB | `output_limit` | 结果的前一部分，并标明已截断 |

`unknown` 这一行是关键的选择。请求发出后连接断开，Mino 无法判断服务是否收到了它。重试一次文档查询也许无害，重试一个会创建工单或发送消息的工具就可能有害，而 Mino 区分不了两者。所以它从不重发调用，连 SDK 自己的重试也关闭了。模型收到 `unknown`，会话随后遵循[第三章的规则](./03-tools-and-bash.md#_4-为行动留下能够如实恢复的记录)：发送你的下一个问题之前，Mino 会请你确认这个结果未知、调用可能已经执行。拒绝确认时问题不会发出；不需要重启。代价是一次打断，哪怕那次调用本来重复执行也无妨。

回放遵循同样的原则。已记录的 MCP 调用只按参数形状校验，所以用过 Context7 的会话，在服务离线或已被删除时仍能打开，也不会重新调用任何东西。每次 MCP 调用都计入每轮 16 次工具调用的既有上限，其结果和其他工具输出一样计入上下文预算。与第七章的历史兼容性见[准备说明](../getting-started.md#第八章的历史兼容性)。

## 5. 用模拟测试检查路由

在 `chapter-08` 的仓库根目录运行：

```bash
go test ./internal/mcp
go test ./internal/agent -run MCP
go test ./internal -run MCPCommand
```

这些测试在测试进程内启动 MCP 服务，分别通过本地 HTTP 和 stdio 子进程通信；Agent 测试还使用本地模拟 Responses 服务。它们不需要 API 密钥，不产生付费请求，也不访问 Context7，不过本地服务需要绑定端口的权限。通过时，每个包都会报告 `ok`。

Agent 测试沿用本章的场景。模拟模型调用 `mcp_docs_query-docs`；测试批准它，检查 `Server`、`Destination`、`Tool` 和 `Arguments` 字段，并确认服务恰好收到一次调用。模型随后再次请求同一工具，测试拒绝，服务的调用计数必须仍为一。最后，测试在完全没有 MCP 工具可用的情况下重新打开会话：保存的问答必须仍能加载。

`internal/mcp` 的测试覆盖边界：配置校验与私有文件权限；暴露名称，包括带哈希后缀的名称；带 bearer 请求头的发现和调用，且请求头不出现在确认界面中；凭证被拒绝时给出 OAuth 提示，且不回显凭证值；超限和中断的调用记为 `unknown`；不跟随重定向；一个 stdio 服务正常工作，同时一个坏掉的服务和一个离线服务被报告而不影响它，退出时 stdio 服务被回收。命令测试检查 `mino mcp add` 写出 `0600` 文件、拒绝非法输入，以及 `mino mcp list` 隐藏请求头的值。

这些测试确认的是程序行为：什么只在批准后发送、确认界面显示什么、失败如何分类，以及服务缺失时历史仍然可用。它们不能证明真实模型会选择 MCP 工具，也不能证明某个第三方服务会一直保持相同的工具、限制或条款。第 1 节的冒烟运行说明 Mino 曾经成功访问 Context7，并不构成兼容性保证。

## 6. 本章小结与下一步

开头那个 React 问题，现在可以取到最新文档了：Mino 在启动时发现了 Context7 的工具，模型请求了其中一个，而参数只在你批准那次调用之后才离开你的机器。凭证不进入对话，结果不确定的调用从不重试，服务消失后会话依然可用。Mino 做不到的，是在服务收到调用之后控制它，以及支持需要 OAuth 的服务。

现在保护措施分散在好几处：Bash 批准、文件工具的范围限定、MCP 目的地展示、未知结果确认。[第 09 章：用安全防护让 Agent 在授权范围内行动](./09-guardrails.md)把这些保护统一成一套由代码执行的策略。
