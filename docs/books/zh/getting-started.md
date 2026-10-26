---
prev:
  text: 关于这本书
  link: /zh/
next:
  text: 01 与模型对话
  link: /zh/chapters/01-terminal-chat
---

# 准备与安装

本页准备于 2026-10-26 发布的 `chapter-11`（程序版本 `0.11.0`），同日 `chapter-10` 重新发布；覆盖本书到第 11 章为止的内容，各章新增的能力见[章节规划](./plan-todo-chapters.md)。运行各章实验时，请使用该章注明的标签。

第二章把[一段对话保存在本地](#第二章的本地历史)，并在启动时恢复。

第三章加入[获批的 Bash 执行](#从源码体验第三章)。新记录采用版本 2，`chapter-02` 无法读取。

第四章[如实恢复中断的工具问答](#从源码体验第四章)，可能要求你先确认结果未知的操作，才能继续聊天。

第五章加入[多会话](#从源码体验第五章)。创建第二个会话前，请阅读[第五章的会话存储](#第五章的会话存储)：更早的版本只保留一段对话。

第六章默认使用 **128,000 tokens** 的上下文窗口，支持[配置覆盖](#配置上下文窗口)。升级已有会话前，请阅读[历史兼容性说明](#第六章的历史兼容性)：新记录采用版本 3。

第七章加入[项目 Skills](#从源码体验第七章)。升级已有会话前，也请阅读[第七章的历史兼容性](#第七章的历史兼容性)：新记录使用版本 4。

第八章加入[项目文件工具](#从源码体验第八章)。在已有会话上使用前，请阅读[第八章的历史兼容性](#第八章的历史兼容性)。

第九章加入 [MCP 工具](#从源码体验第九章)。在已有会话上使用前，请阅读[第九章的历史兼容性](#第九章的历史兼容性)。

第十章加入[安全防护](#从源码体验第十章)：每次工具调用都经过同一个决定，内置受保护路径，并可用 `/mode` 选择[权限模式](#第十章的权限模式)。

第十一章加入[评测](#从源码体验第十一章)：`mino eval` 让声明好的用例经过 Agent 循环并给结果评分。它沿用记录版本 4，见[历史兼容性说明](#第十一章的历史兼容性)。

第十二章正在开发中，在 macOS 上为 Bash 加入[操作系统沙箱](#从源码体验第十二章)。它沿用记录版本 4；用旧版本打开第十二章的会话之前，请阅读[历史兼容性说明](#第十二章的历史兼容性)。

## 准备一台 Mac

支持 macOS 13 及以上，提供 Apple Silicon 和 Intel 两种安装包。安装器会自动选择对应的版本，不需要你先安装 Go。

仓库和发布包现已公开。安装使用 macOS 自带的 `curl`，不需要 GitHub CLI，也不需要登录 GitHub。首次启动配置模型服务时，才需要填写模型 API Key。

## 一行安装

在 Bash 或 zsh 中运行：

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-11 && export PATH="$HOME/.mino/bin:$PATH"
```

此命令从 `main` 获取当前安装器，并选择 `chapter-11`。省略 `-- chapter-11` 即可安装最新发布版。安装器从同一个已确定的发布标签下载 macOS 安装包和校验文件。

安装器校验下载包的 SHA-256 和程序版本后，才替换 `~/.mino/bin/mino`，并设置终端的查找路径。首次安装会创建 `~/.mino`，配置文件会在首次启动填写完成后创建。

## 第一次启动

在任意目录运行：

```bash
mino
```

程序只询问缺失的配置项，所有界面提示均为英文：

```text
API URL [https://api.openai.com/v1]:
Model:
API Key (input hidden):
```

API 地址可以直接回车使用 OpenAI 官方地址。**模型没有默认值**，需要填写服务支持且账号有权限使用的模型名；密钥也需要填写，输入时不显示字符。

自定义服务必须支持 Responses API 流式输出：接受 `stream: true` 请求并返回 `text/event-stream`。Mino 会拒绝仅返回完整 JSON 的端点。填写 API 前缀，例如 `https://gateway.example.com/v1`，不要加上 `/responses` 或 `/chat/completions`。远程地址必须使用 HTTPS。

Mino 会发送有序的 `input` 列表，并请求 `reasoning.encrypted_content`。兼容服务需要接受这些字段，以及它返回后又被程序重放的响应项。第三章及之后的版本还要求服务[支持 Responses 函数工具](#从源码体验第三章)。如果纯文本响应在成功完成时没有提供输出项，Mino 会用已完成的流式文本保存回答。

设置保存在 `~/.mino/config.json`。之后配置齐全就直接进入聊天；升级不会覆盖这些设置。密钥以明文保存在本地，目录和文件分别使用 `0700`、`0600` 权限。

修改服务、模型或密钥时，编辑这个文件后重启。空字段会重新询问；设置过程中按 Ctrl+C，已有配置保持不变。Mino 不读取项目内配置、`.env` 或 `OPENAI_*` 环境变量。不要把配置文件放进仓库。

在 `You>` 后输入问题即可。真实问答会请求你配置的服务，并按该服务的规则计费。只输入 `/exit` 可以验证程序能够启动，无需调用模型。

## 查看版本和更新

如果 Mino 安装于章节标签迁移之前，请先把上面的安装命令**重新运行一次**，即使显示的数字版本没有变化。旧的内嵌更新器无法解析 `chapter-*` 标签。重新安装会保留配置、自定义 `~/.mino/SOUL.md` 和已有历史。

安装章节标签对应的发布版后，可以检查数字版本，再次选择本章：

```bash
mino version
mino update chapter-11
```

不带参数的 `mino update` 选择最新发布版。新版安装器也接受 `0.11.0` 或 `v0.11.0`，两者都指向 `chapter-11`。下载或校验失败时保留现有程序。2026-10-21 之前安装的版本沿用旧的章节编号，见[重新发布的版本](./releases.md#重新发布的版本)；版本号规则见[发布与补丁规则](./releases.md#版本规则)。

## Mino 的身份

配置齐全后，Mino 在进入聊天前读取一次 `~/.mino/SOUL.md`。文件缺失时，程序用内嵌的默认身份创建它。已有内容会在重启和更新时保留；配置未完成或被取消时不会创建这个文件。

默认内容把 Mino 描述为终端助手，可以帮助回答问题、解释概念、写作，以及讨论用户提供的代码，并要求模型使用你的语言、如实说明限制。修改用户文件可以调整这些指引，重启 Mino 后生效；程序不支持即时重载。完整文本会通过 `instructions` 发送给配置的模型服务，不要在其中放置密钥。

第二章的默认身份说明 Mino 会提供对话历史。第三章的默认身份还说明配对工具结果和获批后的 Bash 执行。第四章的默认身份进一步说明：结果未知不算成功，结果未知的操作绝不能自动重试。Mino 会保留已有身份文件。切换章节时，只更新与将要运行的版本不符的能力描述：第二章结束了独立问答，第三章才加入工具。保留其他自定义指令，编辑后重启。

文件必须是普通文件，包含非空的 UTF-8 文本，大小不超过 64 KiB。目录、符号链接或无效文本会在发送模型请求前被拒绝。目录和文件权限分别为 `0700`、`0600`。Mino 忽略工作目录中的 `AGENTS.md` 和 `SOUL.md`：仓库的 `AGENTS.md` 用于开发，根目录的 `SOUL.md` 仅提供构建时嵌入的默认内容。

## 第二章的本地历史 {#第二章的本地历史}

加载配置和身份后，Mino `chapter-02` 会打开 `~/.mino/sessions/` 中唯一保存的对话，恢复已完成的问答。首次启动时，程序创建这个目录和一个空的 `<session_id>.jsonl` 文件，会话 ID 由 32 个小写十六进制字符组成；之后每次启动都打开同一个文件。记录通过 `turn_id` 关联每轮的输入、回答和结束状态；记录中没有 `session_id`，因为文件名已经标明它属于哪段对话。Mino 在查看目录之前，还会先打开固定的 `~/.mino/sessions.lock`，确保这段对话同时只能由一个进程使用。如果启动时提示历史正在使用，请关闭另一个 Mino 进程；退出后锁文件可以继续存在，不应通过删除它绕过仍在运行的进程。

会话文件和锁文件使用 `0600` 权限，`~/.mino/sessions/` 和用户目录使用 `0700`。Mino 拒绝符号链接和非普通的会话或锁文件，也拒绝本身是文件或符号链接的 `sessions`。对话文本以明文保存在本地；程序不会把配置密钥或原始 API 错误复制进历史。你提交的文本会被保存，其中也可能包含你主动输入的敏感数据。不要把历史及恢复副本放进 Git、截图或共享日志。

只有已完成的问答会随后续问题发送。修改配置中的模型或服务不会新建历史：后续请求会把这些已完成的问答发送给新配置的服务。`store: false` 关闭的是服务端响应对象存储，不会关闭本地文件，也不会阻止上下文传输。

如果启动时修复了不完整或 JSON 语法损坏的最后一行，程序会报告会话文件旁边的 `<session_id>-recovery-*.jsonl` 备份；它在修复前保存，权限为 `0600`。尚未结束的一轮会被标记为中断，不会自动重试。中间记录损坏、未知字段或版本，以及错误的记录顺序都会阻止启动，历史内容保持不变；应使用已确认有效的备份恢复，不要随意删除记录。

这个版本只保留一段对话。如果 `~/.mino/sessions/` 中有不止一个以会话 ID 命名的文件，启动会停止并提示 `~/.mino/sessions contains more than one conversation. This release keeps one conversation; use a release that supports multiple sessions.`，不改动任何文件。恢复备份和其他文件名不计入其中。请把多余的文件移到别处，或者使用支持多会话的 `chapter-05` 及之后的版本。

历史限制为每条记录 16 MiB、整个文件 64 MiB。这是文件限制，不是模型的上下文预算。触及限制时，Mino 会报错，不会悄悄丢弃旧记录。如果需要手动重新开始，请先关闭 Mino，保留私有备份，再把会话文件移出 `~/.mino/sessions/` 后重启；下次启动会创建一段新的空对话。`chapter-02` 尚未实现 `/new`、`/clear` 和上下文压缩。

2026-10-21 之前发布的版本把这段对话保存在 `~/.mino/history.jsonl`。当前各版本不会读取或迁移这个文件，请把它作为私有备份保留。

## 从源码体验第三章

第三章已随 `chapter-03` 发布（程序版本 `0.3.0`）。若要跟随源码学习，请按[下面的检出步骤](#从源码学习)选择 `chapter-03`，再在仓库根目录运行：

```bash
git checkout chapter-03
go run ./cmd/mino
```

启动横幅标明 `Chapter 03: Tools and the Agent Loop`，源码构建的版本显示为 `dev`。批准命令需要交互式终端，从文件或管道重定向的输入不能批准执行。每条命令都从启动 Mino 的目录开始，程序将其解析为真实绝对路径，并在批准界面显示。

配置的模型和服务必须支持 Responses 函数工具、严格参数结构、`parallel_tool_calls: false`，以及把 `function_call`、`function_call_output` 与消息、返回的加密推理内容一起重放。仅支持文本流式输出还不够。本机模拟测试验证请求结构，但不证明你的服务商已经兼容。

升级时请检查 [Mino 的身份](#mino-的身份)：已有的 `~/.mino/SOUL.md` 指令会保留。新版默认身份已经说明工具调用、不可信工具输出和执行限制。

Bash 启动时不读取配置文件。程序提供的环境只有 `HOME`、`LANG=en_US.UTF-8` 和下面固定的 `PATH`：

```text
/opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin
```

安装在其他目录中的程序可能需要使用可执行文件的绝对路径。Mino 不继承你的 Shell 别名或其他环境变量，也不把 API 配置复制进子进程环境。**获批的 Bash 仍有你账户的文件和网络访问权限**，包括账户可读取的配置文件；这不是沙箱。批准前，应查看完整命令和显示的环境。

同一个会话文件 `~/.mino/sessions/<session_id>.jsonl` 现在还会保存模型工具调用、开始和结果。第三章能够读取第二章的 `v: 1` 记录，新记录写为 `v: 2`，不会重写旧记录。运行第三章发布版前，请私下备份会话文件。一旦追加了版本 2 记录，`chapter-02` 就不能再读取这份文件：使用旧程序前，应关闭 Mino，另行保留新版文件，再恢复升级前的备份。不要把版本 2 记录直接改标为版本 1。

与第二章一样，第三章只重放已完成的问答。启动恢复绝不重跑命令：未结束的工具问答会在磁盘上补上 `not_executed` 或 `unknown` 结果，并以提示告知你，但它不会进入上下文；失败或被取消的工具问答同样如此。详见[第三章的重启行为](./chapters/03-tools-and-bash.md#_4-2-重启时留在上下文之外的工具问答)。

命令和采集结果会以明文保存，并作为上下文发送到后续请求。它们可能包含敏感内容，包括命令从文件中读到的数据。切换配置中的服务，也会把这些可重放的历史发送给新服务。请妥善保管历史和恢复副本。

## 从源码体验第四章

第四章已随 `chapter-04` 发布（程序版本 `0.4.0`）。请按[下面的检出步骤](#从源码学习)选择该标签，再在仓库根目录运行：

```bash
git checkout chapter-04
go run ./cmd/mino
```

启动横幅标明 `Chapter 04: Recovering Tool Turns`，源码构建的版本显示为 `dev`。Go、模型服务和 Bash 的要求与[第三章说明](#从源码体验第三章)相同。运行这个工作区会使用真实用户目录中的配置和会话文件。

第四章会把失败、取消或中断的工具问答连同配对结果和运行时提示一起重放。如果会话中有尚未确认的 `unknown` 结果，启动时会在横幅之前询问 `Do you understand that these commands may already have run? [y/N]`。请在交互式终端中回答；重定向或管道输入无法确认，`y` 或 `yes` 以外的回答都会退出且不保存确认。在 `recovery_ack` 记录保存之前，每次启动都会再次询问。确认前请先检查命令可能造成的影响，详见[第四章](./chapters/04-recovering-tool-turns.md)。

新版默认身份补充说明：结果未知不算成功，结果未知的操作绝不能自动重试。已有的 `~/.mino/SOUL.md` 会保留；如需这条说明，请自行修改并重启。

第四章仍写 `v: 2` 记录，能直接读取第三章的会话文件。第三章排除在上下文之外的工具问答现在会被重放，第三章留下的 `unknown` 结果会触发确认询问。一旦写入 `recovery_ack` 记录，`chapter-03` 不认识这种记录，会拒绝打开文件。在这两个版本之间切换前，请关闭 Mino，并私下备份 `~/.mino/sessions/`。

## 从源码体验第五章

第五章已随 [chapter-05](https://github.com/qshine/mino/releases/tag/chapter-05) 发布，程序版本为 `0.5.0`，安装包包含多会话功能。若要运行对应源码，请在仓库根目录选择该标签：

```bash
git checkout chapter-05
go run ./cmd/mino
```

启动横幅标明 `Chapter 05: Multiple Sessions`，源码构建的版本仍显示为 `dev`。Go 和模型服务要求与[源码准备说明](#从源码学习)、[第三章工具说明](#从源码体验第三章)相同。运行这个工作区会使用真实用户目录中的配置，以及前几章保存的对话。体验前，请关闭其他 Mino 进程，并私下备份 `~/.mino/sessions/`。

用 `/new` 开始独立对话，`/sessions` 列出 ID，`/resume <id>` 选择由 32 个小写十六进制字符组成的完整 ID，`/help` 查看命令。`/clear` 需要在交互式终端中准确输入当前会话 ID，`/exit` 退出。这些命令不会调用模型。如果启动或 `/resume` 时遇到未知工具结果，需要先完成第四章的恢复确认才能继续。[第五章](./chapters/05-jsonl-sessions.md)展示了 A → B → A 实验和清空确认行为。

切换会话不会重新加载 `~/.mino/SOUL.md`，也不会改变启动时确定的 Bash 目录。已有身份文件保持不变；如果其中仍称 Mino 只有一段对话，请自行更新这条指引并重启。所有会话共用已加载的模型配置：修改服务并重启后，下一次提问会把选中会话的可重放历史发送给该服务。

## 第五章的会话存储

第五章在 `~/.mino/` 中使用以下布局：

```text
sessions/<session_id>.jsonl
active-session.json
sessions.lock
```

用户目录和会话目录使用 `0700`，会话日志、选择文件和锁使用 `0600`。程序打开它们时会拒绝符号链接和非预期文件类型。会话内容仍是明文，可能含有私人问题、命令和工具输出。不要把会话文件或恢复副本放进 Git 和共享日志。每份 JSONL 仍限制为单条记录 16 MiB、整个文件 64 MiB；它们不是模型上下文预算。程序仍读取 `v: 1`、`v: 2` 记录，新记录写为 `v: 2`。

Mino 在整次运行期间持有 `sessions.lock`，因此同一个会话库只能由一个 Mino 进程使用，即使你想选中不同会话也一样。第二至第四章使用同一把锁，所以仍在运行的旧版 Mino 也会阻止启动。应关闭另一个进程，而不是删除它的锁；退出后锁文件可以继续存在。

首次使用时，程序新建会话；已有活动选择时，恢复对应会话。没有 `active-session.json`、而 `sessions/` 中恰好有一段对话时（第二至第四章留下的正是这种状态），Mino 原地选中该文件并保存选择，不复制也不改写文件。有多个会话却缺少选择、选择已损坏，或选中日志不可用时，Mino 会等待你重新选择。先用 `/sessions` 查看，再用完整的已有 ID 执行 `/resume`，也可以选择 `/new`。列表显示 ID 和更新时间，不代表日志已经通过校验，加载时仍会检查。修复不完整尾部时，程序在 `sessions/` 中保存私有的 `<session_id>-recovery-*.jsonl` 备份。中间损坏或无效记录会导致日志被拒绝，不会悄悄丢弃内容；恢复写入失败则停止程序。

第二至第四章只支持一段对话，也不读取 `active-session.json`。一旦 `/new` 创建了第二个会话文件，它们会在启动时以 `~/.mino/sessions contains more than one conversation` 停止。若要回退到旧程序，请先关闭 Mino，私下复制一份 `~/.mino/sessions/`，并在该目录中只保留你想继续的那段对话。

`/clear` 保留当前会话 ID 和其他会话，确认后将活动日志替换为空历史。它不删除恢复副本或手工备份，也不会撤销命令或安全擦除磁盘数据。保存或同步会话变更可能失败，Mino 此时停止聊天，不会承诺成功。请保留文件并检查错误提示，再重启让程序重新核对保存状态。

## 从源码体验第六章

第六章已随 [chapter-06](https://github.com/qshine/mino/releases/tag/chapter-06) 发布，程序版本为 `0.6.0`。若要运行对应源码，请在仓库根目录选择该标签：

```bash
git checkout chapter-06
go run ./cmd/mino
```

启动横幅标明 `Chapter 06: Context Compaction`，源码版本仍显示为 `dev`。已有的 [Go 准备说明](#从源码学习)和 [Responses 工具要求](#从源码体验第三章)继续适用。本版还要求服务接受 `max_output_tokens`、`truncation: "disabled"`，以及带 `tool_choice: "none"` 的无工具摘要请求。本机模拟测试验证这些字段，但不证明服务商已经兼容。

在下一个 `You>` 提示后输入 `/compact`，程序会整理选中会话的较早轮次，完整保留最近两轮可重放问答。自动预算检查在请求前使用相同流程，也包含工具返回后的请求。与会话选择命令不同，`/compact` 可能调用模型并产生费用；没有较早轮次时不发送请求。交互过程、限制和测试见[第六章](./chapters/06-context-compaction.md)。

已有 SOUL 文件保持不变。如果其中仍称 Mino 总是提供完整历史，或不具备摘要能力，请更新这些能力描述，同时保留自定义指引，编辑后重启。摘要是有损上下文，不是新的指令或工具授权。

### 配置上下文窗口

第六章在 `~/.mino/config.json` 中增加可选字段 `context_window`，单位为 token，必须是非负整数。省略或填写 `0` 时使用 **128K（128,000 tokens）**，正整数覆盖默认值。负数、小数、字符串和 `null` 会被拒绝。首次设置不会询问这个可选字段。

例如，如果部署实际支持 64,000-token 窗口，可以把下面的字段加入现有 JSON 对象，然后重启。这是字段示意，不是完整配置；请保留已有的 `base_url`、`api_key` 和 `model`。

```json
{
  "context_window": 64000
}
```

启动时会显示 `Context window: 64000 tokens (config).`；字段省略或为零时显示 `Context window: 128000 tokens (default).`。这些是本地设置，启动不会请求模型元数据，配置值也不能扩大服务的真实容量。请按模型和部署实际支持的窗口填写。

Mino 会预留输出空间，并按字节估算输入，因此这个值既不是文件大小限制，也不是已消耗 token 的精确计数。值设得过小，可能连正常问题也被拒绝；设得过大，仍可能遇到服务端错误。[预算说明](./chapters/06-context-compaction.md#_3-每次请求前检查预算)解释了触发阈值，以及请求过大或压缩失败时的处理。

### 第六章的历史兼容性

在已有对话上运行本章版本之前，请关闭其他 Mino 进程，并私下备份升级前的会话日志与 `active-session.json`。第六章读取记录版本 1–3，新记录统一写为 `v: 3`，包括普通问答。一旦追加新记录，旧发布版就无法加载受影响的日志，即使你还没有使用 `/compact`。不要把版本 3 记录直接改标为版本 2。

如需回退到旧程序，请先关闭 Mino，另行保留新版文件，再恢复配套的升级前文件。摘要保存在选中会话既有的 JSONL 文件中，没有需要单独复制的摘要文件。会话选择和恢复仍按[第五章的会话存储](#第五章的会话存储)执行。

压缩只追加摘要，不删除原始消息，因此不会减少磁盘占用或擦除敏感内容。提交给摘要请求的较早材料，以及生成的摘要，都可能包含私人信息。请妥善保管会话文件和恢复副本。修改配置的服务，也会改变后续摘要和聊天请求的接收方。

## 从源码体验第七章

第七章已随 [chapter-07](https://github.com/qshine/mino/releases/tag/chapter-07) 发布，程序版本为 `0.7.0`。若要运行对应源码，请在仓库根目录选择该标签：

```bash
git checkout chapter-07
go run ./cmd/mino
```

横幅标明 `Chapter 07: Project Skills`，源码版本为 `dev`。既有的 Go、Responses 工具与压缩要求继续适用。这次运行会使用用户目录中的配置与会话；使用已有对话前，请阅读[第七章的历史兼容性](#第七章的历史兼容性)。不产生模型费用的模拟检查见[本章实验](./chapters/07-skills.md#_4-检查请求-而不只听回答怎么说)。

技能发现只扫描启动 Mino 时所在目录下的 `.agents/skills/`，不搜索父目录或用户的 `~/.agents/skills/`。新会话需要的文件应在启动前准备好；如果 Mino 恢复了已初始化的会话，用 `/new` 发现这些文件。空目录也会保存为快照，因此后来添加文件不会更新该会话。

### 编写项目技能

在项目目录下创建 `.agents/skills/review/SKILL.md`，写入以下示例内容：

```text
---
name: review
description: 结合测试和具体证据审查代码改动。
---
# 审查改动

阅读提供的改动及其测试。每项发现都要指出具体的失败情形，
区分已确认的问题与不确定之处。需要更多证据时，请求补充；
没有测试结果时，不要声称测试已经通过。
```

在 `You>` 输入“用 review 技能审查这处改动”，并提供一处小改动。模型可以用 `{"name":"review"}` 请求 `skill_view`。读取无需确认，之后的 Bash 命令仍要单独批准。Mino 把完整文件显示为工具结果，再由模型继续处理。

只使用直接子目录，发现不会递归。YAML 头部以 `---` 开始和结束，结束分隔符须位于前 16 KiB 内。`name` 必须与目录名一致，由小写 ASCII 字母或数字组成，可用单个连字符分隔，最多 64 个字符。`description` 必须是非空 YAML 字符串，最大 1,024 个 UTF-8 字节，支持引号字符串和块字符串。其他元数据不进入目录。

每个 `SKILL.md` 必须是最大 64 KiB 的普通文件；读取时要求内容是非空 UTF-8 文本且不含 NUL。包含项目路径的序列化目录最大 32 KiB。技能目录不存在时得到空目录；没有 `SKILL.md` 的直接子目录，以及符号链接子目录会被忽略。候选文件格式错误、文件为符号链接或特殊文件，以及 `.agents/skills` 路径中的目录为符号链接时，都会被拒绝。发现错误会阻止初始化；`/new` 失败时保留原先选中的会话。

名称和简介在会话内保持固定，包括 `/resume`、重启与确认 `/clear` 之后；只有 `/new` 发现新目录。`skill_view` 从已保存的项目位置读取当前正文，只接受目录中的名称，不接受任意路径，不自动读取引用文件或执行脚本。已有 `~/.mino/SOUL.md` 保持原样，每次进程启动读取一次；其中的过时能力说明需要你自行更新，再重启生效。

### 第七章的历史兼容性

在已有对话上运行第七章前，关闭 Mino，私下备份会话日志与 `active-session.json`。本版读取记录版本 1–4，所有新记录统一写为 `v: 4`。尚无目录的会话在本版首次初始化时，就会持久化 `session_skills` 快照，不必等到下一次提问。空快照也使用版本 4，因此仅启动聊天就可能让旧发布版无法再读取选中的日志。

回退时，先停止 Mino，另行保留新版文件，再恢复对应的升级前备份。不要直接修改版本 4 记录的版本号。`/clear` 删除选中会话的对话和摘要，同时保留技能快照，不能用来降低格式版本。

目录会把启动时确定的项目路径、技能名称与简介发送给配置的模型服务。请求读取的完整文件会作为工具结果，进入会话日志与模型上下文。技能中不要放入秘密信息，会话日志也应保持私有；这些内容还可能进入后续摘要请求。固定元数据、读取当前文件，以及缓存相关验证的边界，见[第七章](./chapters/07-skills.md)。

## 从源码体验第八章

第八章已随 [chapter-08](https://github.com/qshine/mino/releases/tag/chapter-08) 发布，程序版本为 `0.8.0`。在项目目录启动已安装的 `mino` 即可使用文件工具。若要运行对应源码，请在仓库根目录选择该标签并构建，再在你希望它处理的项目里启动：

```bash
git checkout chapter-08
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

把两个路径替换成你自己的目录。横幅显示 `Mino - Chapter 08: Project File Tools`，源码版本为 `dev`，启动提示以 `Bash commands and file writes require approval; project file reads do not.` 结尾。既有的 Go、Responses 工具、压缩与技能要求继续适用。不产生模型费用的模拟检查见[本章实验](./chapters/08-file-tools.md#_4-用模拟测试检查批准流程)。

文件工具只在启动 Mino 的目录内工作，也就是 Bash 使用的目录；`/new` 和 `/resume` 不会改变它。`read_file` 无需批准，所以它读到的任何内容，包括项目里的 `.env` 文件，都会发送给配置的模型服务并保存到会话日志。请在项目目录而不是用户主目录启动 Mino：如果启动目录包含 `~/.mino`，每个文件工具都会返回错误结果，不读也不写。

已有的 `~/.mino/SOUL.md` 保持原样。新安装使用的默认 SOUL 会告诉模型：处理项目文件时优先使用文件工具而不是 Bash，编辑前先读取文件。如果你的 SOUL 把 Bash 写成唯一的工具，请自行补充类似说明，再重启。

### 第八章的历史兼容性

第八章沿用记录版本 4，没有新增记录类型，因此无需迁移即可打开第七章的会话。反过来则不行：会话中一旦运行过任何文件工具（包括免批准的读取），`chapter-07` 就无法加载它，会报告存在一个针对不可用工具的已开始调用。在已有对话上体验第八章前，关闭 Mino，私下备份会话日志与 `active-session.json`。回退时，先停止 Mino，另行保留新版文件，再恢复对应的备份。

## 从源码体验第九章

第九章已随 [chapter-09](https://github.com/qshine/mino/releases/tag/chapter-09) 发布，程序版本为 `0.9.0`。已安装的 `mino` 已包含 `mino mcp` 和 MCP 工具。若要运行对应源码，请在仓库根目录选择该标签并构建，再在项目目录中启动：

```bash
git checkout chapter-09
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

把两个路径替换成你自己的目录。横幅显示 `Mino - Chapter 09: MCP Tools`，源码版本为 `dev`，启动提示以 `Bash commands, file writes, and MCP tool calls require approval; project file reads do not.` 结尾。既有的 Go、Responses 工具、压缩、技能与文件工具要求继续适用。不产生模型费用的模拟检查见[本章实验](./chapters/09-mcp.md#_5-用模拟测试检查路由)。

### 添加、查看和删除 MCP 服务

在聊天之外用 `mino mcp`（在检出目录中为 `bin/mino mcp`）管理服务。按名称和 URL 添加一个远端 Streamable HTTP 服务：

```bash
mino mcp add context7 https://mcp.context7.com/mcp
```

```text
Added MCP server context7 (https://mcp.context7.com/mcp). It connects the next time chat starts; each tool call asks for approval.
```

如果服务需要 API 密钥，加上 `--bearer` 保存一个 `Authorization: Bearer` 请求头，或用 `--header NAME`（可重复）保存其他请求头。本地 stdio 服务把命令写在 `--` 之后，可选 `--env NAME`（可重复）和 `--cwd DIR`：

```bash
mino mcp add context7 https://mcp.context7.com/mcp --bearer
mino mcp add local --env API_TOKEN -- /path/to/mcp-server
mino mcp list
mino mcp remove local
```

把 `/path/to/mcp-server` 换成你信任的服务程序；每次启动聊天，Mino 都会启动它。添加时，命令会被解析为绝对路径；运行时连同参数直接执行，不经过 shell。工作目录默认为你的用户主目录。服务名称使用小写字母、数字和单个连字符，最多 32 个字符；最多可配置 16 个服务。完整用法见 `mino mcp help`。

请求头和环境变量的值在关闭回显的情况下输入，因此添加它们需要交互式终端。这些值之后不会再显示：`mino mcp list` 只显示名称。例如，用 bearer 令牌添加 Context7 后，它会输出类似下面的一行（交互示意）：

```text
context7  http  https://mcp.context7.com/mcp  headers=Authorization (values hidden)
```

添加服务不会连接它。下次启动聊天时，Mino 会并行连接所有已配置的服务，不再询问，并为每个服务打印一行：`MCP server NAME: N tools from DESTINATION` 或 `MCP server NAME unavailable: REASON`。失败的服务绝不会让聊天停下。之后每次工具调用都会请求批准，详见[本章](./chapters/09-mcp.md)。

### MCP 配置与隐私

配置文件是 `~/.mino/mcp.json`，一个严格的 JSON 文件，权限为 `0600`。如果它是符号链接、其他用户可访问、格式错误或含有未知字段，Mino 会拒绝启动聊天；`mino mcp` 命令同样拒绝读取它，因此需要手动修正。你可以手动编辑这个文件，但其中以明文保存着你的凭证：不要把它放进 Git、分享出去的备份或截图。Mino 从不读取项目目录或环境变量中的 MCP 配置。

凭证只发给对应的服务，从不发给模型服务，也不进入会话日志、确认界面或错误信息。工具参数和结果则是另一回事。你批准的参数会离开本机、发往该服务；结果会和其他工具输出一样保存到会话日志并发送给模型服务。如果参数里有你不希望该服务收到的秘密或私人数据，就不要批准这次调用。

远端 URL 必须使用 HTTPS，只有 `localhost` 和回环地址允许明文 HTTP。不跟随重定向。本章不支持 OAuth 或浏览器登录、旧版 HTTP+SSE 传输，以及 MCP 的 resources、prompts 和 sampling。拒绝静态凭证的服务会被报告为不可用，并说明不支持 OAuth。服务的工具列表只在启动时读取；有变化时请重启 Mino。

已有的 `~/.mino/SOUL.md` 保持原样。新安装使用的默认 SOUL 会介绍 `mcp_` 工具，说明它们的描述和结果不可信，并要求模型不要在参数中放入秘密信息、不要重试结果未知的调用。如果需要，请在自己的 SOUL 中补充类似说明，再重启。

### 第九章的历史兼容性

第九章沿用记录版本 4，没有新增记录类型，因此无需迁移即可打开第八章的会话。已记录的 MCP 调用只按参数形状校验，所以服务离线或已被删除时，会话仍能打开，也不会重新调用任何东西。反过来则不行：会话一旦开始过 MCP 调用，`chapter-08` 就无法加载它，会报告存在一个针对不可用工具的已开始调用。在已有对话上体验第九章前，关闭 Mino，私下备份会话日志与 `active-session.json`。回退时，先停止 Mino，另行保留新版文件，再恢复对应的备份。`chapter-08` 的可执行文件会忽略 `~/.mino/mcp.json`。

## 从源码体验第十章

第十章已随 [chapter-10](https://github.com/qshine/mino/releases/tag/chapter-10) 发布，程序版本为 `0.10.0`，并于 2026-10-26 以权限模式重新发布。若要运行对应源码，请在仓库根目录选择该标签并构建，再在项目目录中启动：

```bash
git checkout chapter-10
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

把两个路径替换成你自己的目录。启动时，`Session:` 那一行下面会打印 `Permission mode: manual. Use /mode to change it.`；`/new` 和 `/resume` 之后也会出现这两行。横幅显示 `Mino - Chapter 10: Guardrails`，源码版本为 `dev`。既有的 Go、Responses 工具、压缩、技能、文件工具与 MCP 要求继续适用。不产生模型费用的模拟检查见[本章实验](./chapters/10-guardrails.md#_7-用模拟测试检查这个决定)。

### 第十章的权限模式

第十章没有权限配置文件。请在聊天中为当前会话选择模式：

| 命令 | 效果 |
| --- | --- |
| `/mode` | 显示当前会话的模式。 |
| `/mode manual` | 每条 Bash 命令、每次文件写入或编辑、每次 MCP 调用之前都询问。每个会话都从这个模式开始。 |
| `/mode accept-edits` | 另外让 `write_file` 和 `edit_file` 不经询问运行。Bash 和 MCP 调用仍会询问。 |
| `/mode bypass` | 所有调用都不经询问运行，包括 Bash 和 MCP。 |

切换到更宽松的模式会打印警告并询问 `Switch to this mode? [y/N]`，而且只能在交互式终端中完成。切换到更严格的模式从不询问。每个会话保留自己的模式，直到 Mino 退出：`/new` 从 `manual` 开始，`/resume` 回到该会话在这次运行中先前的模式，`/clear` 保留模式。模式从不保存，所以重启之后每个会话又都是 `manual`。模式放行、不经询问运行的每次调用，仍会连同字段一起打印出来。`/help` 列出了这个命令。

受保护路径在每种模式下都生效，包括 `bypass`：`.env`、`.env.*`、`*.pem`、`*.key`、`*.p12`、`id_rsa*`、`id_ecdsa*`、`id_ed25519*`、`.netrc`、`.npmrc`、`.pypirc`，以及 `.git/`、`.ssh/`、`.aws/`、`.gnupg/` 目录。文件工具从不读写匹配的路径，也没有可以覆盖这一点的提示或设置。匹配不区分大小写，与默认的 macOS 文件系统一致，所以 `.ENV` 同样受保护。

文件工具还有一处变化：只要路径的任意一段目录经过符号链接，即使链接仍指向项目内部，也会被拒绝。如果你的项目通过链接目录访问文件，请改用真实路径。

如果你在重新发布之前安装了 `0.10.0`，它启动时会打印一行 `Policy:`；请运行 `mino update chapter-10` 替换它。重新发布的版本会忽略 `~/.mino/policy.json`。`"bash": "deny"` 或 `deny_tools` 这样的设置不再生效：所有工具都会重新提供给模型，在 `manual` 模式下每次 Bash、写入和 MCP 调用都会询问。原先由 `allow_tools` 免确认的工具现在也会询问。你可以删除这个文件。

已有的 `~/.mino/SOUL.md` 保持原样。新安装使用的默认 SOUL 介绍了权限模式，并要求模型如实报告拦截，不要设法绕过。如果需要，请在自己的 SOUL 中补充类似说明，再重启。无论 SOUL 怎么写，这个决定都会执行。

模式决定哪些调用需要你回答，受保护路径约束文件工具。两者都不是沙箱：运行起来的 Bash 命令或已配置的 MCP 服务以你的账户权限运行；在 `bypass` 模式下，`cat .env` 这样的 Bash 命令会不经询问运行。这些限制见[本章说明](./chapters/10-guardrails.md#_6-策略看不到的地方)。

### 第十章的历史兼容性

第十章沿用记录版本 4，没有新增记录类型或结果状态，权限模式也从不写入会话。被这个决定拒绝的调用，和你拒绝的调用保存方式相同：模型的请求，后面跟着一个 `denied` 结果。它无需迁移即可打开第九章的会话，`chapter-09` 也仍能打开第十章写入的会话。不过回退会让安全防护失效：`chapter-09` 没有受保护路径，它的 `read_file` 会不经询问读取 `.env`。和之前一样，切换版本前请私下备份会话日志与 `active-session.json`。

## 从源码体验第十一章

第十一章已随 [chapter-11](https://github.com/qshine/mino/releases/tag/chapter-11) 发布，程序版本为 `0.11.0`，发布于 2026-10-26；已安装的 `mino` 已经包含 `mino eval`。若要运行对应源码，请在仓库根目录选择该标签并构建，再在同一位置运行仓库自带的用例：

```bash
git checkout chapter-11
go build -o bin/mino ./cmd/mino
bin/mino eval ./evals
```

聊天横幅显示 `Mino - Chapter 11: Evals`，`mino help` 列出 `eval`，`mino eval help` 打印用法。默认的回放模式用脚本化的本机模型驱动 Agent 循环：不需要 API 密钥和网络，也从不读取 `~/.mino`，因此碰不到你的配置和会话。每份用例在全新的临时工作区和主目录中运行，结束后删除。既有的 Go 要求继续适用；正文见[第十一章](./chapters/11-evals.md)。

| 参数 | 含义 |
| --- | --- |
| `PATH` | 一个用例文件，或一个目录，其中的 `*.yaml` 与 `*.yml` 文件按文件名顺序读取，不递归。默认 `./evals`。 |
| `--live` | 使用 `~/.mino/config.json` 中的模型，并以 `~/.mino/SOUL.md` 作为默认指令；文件缺失时像聊天一样先用内嵌默认身份创建它。配置必须完整，命令不会提示输入。标记为 `live: false` 的用例会被跳过，并在表格之前用一行列出名称。实时运行会产生真实请求费用。 |
| `--runs N` | 每份用例的运行次数，1 到 32。默认 1，带 `--live` 时默认 3。用例自己的 `runs` 字段会覆盖它。 |
| `--filter NAME` | 只运行指定名称的用例。 |
| `--out DIR` | 写出 `report.json`，并把每次运行的会话日志保存为 `DIR/<case>/run-N.jsonl`。该目录不能位于 `~/.mino` 之内。 |
| `--timeout DURATION` | 单次运行的时限，1s 到 30m。默认 2m。 |

只要有任何用例的任何一次运行失败，退出状态就是 1。每次运行都从 `manual` 权限模式开始，且不连接任何 MCP 服务，无论你的 `~/.mino/mcp.json` 怎么写；`/mode` 是聊天命令，从不作用于评测运行，`.env` 等受保护路径也和聊天中一样被拒绝。用例批准 Bash 时，它在临时工作区里以你的账户权限运行；这个框架不是沙箱。

### 编写评测用例

在项目中创建 `evals/hello-file.yaml`。文件名去掉扩展名后必须等于 `name`：

```yaml
name: hello-file
description: An approved write creates the file the user asked for.
prompt: Create hello.txt containing hi.
approve: [write_file]
model:
  - call: {tool: write_file, arguments: {path: hello.txt, content: "hi\n", mode: create}}
  - text: hello.txt now contains hi.
graders:
  - tool_order: [write_file]
  - result_status: {call: 1, status: completed}
  - file_contains: {path: hello.txt, text: hi}
  - max_requests: 2
```

它的结构仿照仓库中的 `write-then-read.yaml`。用 `bin/mino eval --filter hello-file ./evals` 单独运行它；在第十一章源码上使用临时主目录观察到的结果是一行通过的记录，两次请求、一次工具调用，运行后临时主目录中仍然没有 `.mino` 目录。

`name` 和 `prompt` 必填，`prompt` 最大 16 KiB。`name` 由小写 ASCII 字母和数字组成，可用单个连字符分隔，最多 64 字节。`approve` 为 `all`（默认）、`none`，或一个要批准的工具名列表。`instructions` 替换这份用例的默认指令。`workspace.files` 在每次运行前预置相对路径的文件，最多 64 个、每个最多 64 KiB。`model` 是最多 16 步的回放脚本，每步是 `call: {tool, arguments}` 或 `text`，可附带 `expect_input_contains`，即该步模型收到的请求体中必须出现的子串；只有使用 `--live` 运行时才能省略它。`live: false` 把用例标记为仅回放，即它断言的是真实模型不会走的脚本路径；`--live` 会跳过它，而且它必须带有 `model` 脚本。省略这个字段表示用例也参与实时运行。`graders` 包含 1 到 32 项检查，必须全部通过：`final_text_contains`、`final_text_not_contains`、`final_text_matches`（不超过 1 KiB 的 RE2 正则）、`tool_called`、`tool_not_called`、`tool_order`、`result_status`（`call` 从 1 开始，按模型请求的全部调用计数）、`file_exists`、`file_contains`、`max_requests` 和 `max_tool_calls`。打算实时运行的用例应当评判真实模型能够到达的结果，例如已写入的文件或没有出现的密钥，而不是某一个短语。未知字段导致加载失败，一个目录最多 256 份用例，一份用例文件最大 64 KiB。

用例可以调用的工具就是聊天工具去掉 MCP：`bash`、`read_file`、`write_file`、`edit_file`，以及读取工作区 `.agents/skills` 下预置技能的 `skill_view`。用例文件和项目技能一样，是你主动运行的项目数据，从不从 `~/.mino` 加载。

### 第十一章的历史兼容性

第十一章沿用记录版本 4，没有新增记录类型或结果状态。运行时唯一的新增是一个只存在于内存中的 `response_completed` 事件，它把 token 用量交给 `mino eval`，从不写入会话日志。因此 `chapter-10` 可以打开第十一章写入的会话，第十一章也无需迁移即可打开第十章的会话；`chapter-10` 唯一缺少的是 `mino eval` 命令。评测运行只把日志写到临时目录，或写到你用 `--out` 指定的目录，从不写入 `~/.mino/sessions`。

## 从源码体验第十二章

第十二章正在开发中。它的发布标签 `chapter-12`（程序版本 `0.12.0`）正在准备，所以暂时没有可安装的发布包；在标签发布之前，请在仓库根目录构建当前源码，再到一个项目目录里启动它。标签发布后，`git checkout chapter-12` 选择的就是同一份代码。

```bash
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

把两个路径替换为你自己的目录。横幅显示 `Mino - Chapter 12: Sandbox`，源码版本为 `dev`。既有的 Go、Responses 工具、压缩、Skills、文件工具、MCP 和安全防护要求继续适用；正文见[第十二章](./chapters/12-sandbox.md)，免费的检查见[第 5 节](./chapters/12-sandbox.md#_5-用评测和测试检查它)。

在带有 `/usr/bin/sandbox-exec` 的 macOS 上，每条 Bash 命令默认受限运行：可以读取受保护路径之外的文件，只能写入项目目录、系统临时目录、`~/Library/Caches` 和 `~/.cache`，不能写入 `.git/`，也不能建立网络连接。每次 Bash 确认，以及 `accept-edits` 或 `bypass` 下不经询问的 Bash 那一行，都会显示带有这段摘要的 `Sandbox` 字段。模型可以为需要网络或在别处写入的命令（例如 `git commit`）设置 `sandbox: false`；这时 Mino 显示 `Sandbox: "off: the model asked to run this command unconfined."`，并在除 `bypass` 外的每种模式下询问你。因为受限命令只能改动项目、临时和缓存目录，`/mode accept-edits` 现在会像运行 `write_file`、`edit_file` 那样不经询问运行沙箱内的 Bash；`manual` 仍对每条命令询问。沙箱不限制对未受保护文件的读取，不约束 `http` MCP 服务，也不约束你以 `sandbox: false` 批准的命令。

如果没有 `sandbox-exec`，启动时会打印 `Sandbox unavailable on this system: Bash commands run with your account permissions.`，确认界面显示 `Sandbox: "unavailable on this system."`，Bash 的行为与第十一章相同。没有关闭沙箱的参数，本章也没有 Linux 沙箱。

本地 stdio MCP 服务可以用 `mino mcp add NAME --sandbox --cwd DIR -- COMMAND` 限定；`mino mcp list` 会在它那一行显示 `sandbox`。它运行时写入限定在自己的 `cwd`、临时和缓存目录内，受保护路径被阻止，网络允许。要求沙箱却无法受限的服务会在启动时被报告为不可用，而不是不受限地启动；`http` 服务和没有沙箱的系统都会拒绝 `--sandbox`。

评测用例和聊天一样受限运行。`evals/` 中的三份 `sandbox-*` 用例需要沙箱，没有沙箱时会失败；因此 `bash scripts/check.sh` 只在 macOS 上通过。

已有的 `~/.mino/SOUL.md` 保持原样。新安装使用的默认 SOUL 介绍了沙箱，并要求模型只在说明理由时才请求 `sandbox: false`。如果需要，请在自己的 SOUL 中补充类似说明，再重启；无论 SOUL 怎么写，沙箱都会执行。

### 第十二章的历史兼容性

第十二章沿用记录版本 4，没有新增记录类型或结果状态，沙箱的任何信息都不会写入会话。受限命令记录为 `{"command":"..."}`，与第三章到第十一章记录每条命令的方式完全相同，所以第十二章无需迁移即可打开它们的会话，`chapter-11` 也能打开只包含受限命令的第十二章会话。例外是获批的选择退出：它的启动记录保留 `"sandbox":false`，`chapter-11` 会把它当作未知的 Bash 参数而拒绝加载该会话。和之前一样，切换版本前请私下备份会话日志与 `active-session.json`。

## 从源码学习

若要运行或修改源码，需要 Go 1.27.1 或更高的兼容工具链。克隆仓库后，选择 `chapter-11`，在仓库目录运行：

```bash
git clone https://github.com/qshine/mino.git
cd mino
git checkout chapter-11
go run ./cmd/mino
```

应用代码及测试位于 `internal/`。`internal/gateway/` 处理终端交互，`internal/agent/` 负责模型请求和历史保存。第三章在 `internal/tools/` 中加入工具。配置和身份加载保留在 `internal/`，无需代码生成步骤。

模块文件仍在仓库根目录。`go.mod` 将官方 `github.com/openai/openai-go/v3` SDK 固定为 v3.66.0，并将官方 `github.com/modelcontextprotocol/go-sdk` 固定为 v1.8.0，`go.sum` 记录依赖校验值。首次构建时，Go 会下载依赖，无需单独安装 SDK。

安装包和源码运行共用用户目录中的配置，以及保存在 `~/.mino/sessions/` 中的对话。自动化测试使用临时用户目录和模拟模型服务，不改动你真实的 `~/.mino` 文件，也不需要真实 API Key：

```bash
bash scripts/check.sh
```

接下来可以从[第一章：与模型对话](./chapters/01-terminal-chat.md)开始，按顺序阅读各章。请使用每章注明的源码版本运行实验，各版本的请求和历史格式有所不同。
