# Mino

[English](README.md) | 简体中文

**用 Go 从零构建自己的 Agent：从一次交互理解原理，用代码和实验验证能力。**

Mino 是面向 Agent 入门者的分章教程，使用 Go 和 OpenAI Responses API。
每章从一段可以观察的交互开始，看清模型收到什么、程序负责什么，再用小实验检查结果与边界。
阅读源码需要了解 Go 的变量、函数和基本错误处理；第一次接触模型 API 也可以从第 01 章开始。

**在线阅读：**[中文教程](https://qshine.github.io/mino/zh/) · [English book](https://qshine.github.io/mino/)

**当前进度：**本快照包含第 01–15 章，对应发布版本 `chapter-15`（程序版本 `0.15.0`）。
第 01–15 章均已发布，标签为 `chapter-01` 至 `chapter-15`，程序版本对应 `0.1.0` 至 `0.15.0`。
第 01 章实现流式终端问答。
第 02 章加入 JSONL 对话历史和重启恢复。
第 03 章加入逐次批准的 Bash 执行与 Agent 循环。
第 04 章如实恢复中断的工具调用：结果交回模型，结果未知的操作需要你确认。
第 05 章加入独立会话、新建、恢复和确认清空。
第 06 章加入上下文压缩：手动 `/compact`、自动预算检查和摘要持久化。
第 07 章加入项目 Skills：会话内固定的技能目录，以及按需读取正文的 `skill_view`。
第 08 章加入项目文件工具：限定在项目目录内的 `read_file`、`write_file` 和 `edit_file`。
第 09 章加入 MCP 工具：用 `mino mcp` 管理 Context7 等远端和本地 MCP 服务。
第 10 章加入安全防护：由代码执行的统一工具策略、受保护路径，以及用 `/mode` 选择的三种权限模式。
第 11 章加入评测：`mino eval` 用声明式用例驱动 Agent 循环，并用确定性检查评分。
第 12 章加入操作系统沙箱：在 macOS 上，Bash 命令在 Seatbelt 配置下运行，写入受限，网络和受保护路径被阻断。
第 13 章加入用户记忆：每条记忆是 `~/.mino/memory/` 下的一个文件，每轮都会把所有记忆的一行摘要交给模型。
第 14 章加入子 Agent：`spawn_agent` 把最多 10 个委派任务交给并行的子 Agent，只取回它们的最终文本。
第 15 章加入自定义子 Agent：在 `.agents/agents/<name>.md` 中定义角色，带有自己的角色说明和更窄的工具列表，并内置 `create-subagent` 技能。
Gateway 与 Agent 目录从第 01 章起保持一致。
详见[章节规划](docs/books/zh/plan-todo-chapters.md)。

**作者：qqling | AI Builder。**我想从零构建一个属于自己的 Agent，把持续研究和实践中的理解整理成入门教程。
你可以在[作者介绍](docs/books/zh/index.md#关于作者)中了解我的创作初衷，并通过 GitHub、微信公众号或小红书关注和交流。

## 安装

支持 **macOS 13 及以上版本**，提供 **Apple Silicon 和 Intel Mac** 安装包。
仓库和安装包已公开，无需安装 Go、GitHub CLI，也无需登录 GitHub。
安装程序使用 macOS 自带的 `curl` 下载。

在 Bash 或 zsh 中执行这一行命令：

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
```

安装程序会识别 Mac 架构，下载最新发布版本，校验 SHA-256 和程序版本，然后安装到 `~/.mino/bin/mino`。
程序目录会加入 zsh 或 Bash 登录配置；命令最后的 `export` 让当前终端也能立即使用 `mino`。
不需要 `sudo`，已有配置会保留。

## 启动

在任意目录执行：

```bash
mino
```

第一次启动会用英文询问缺失的配置：

```text
API URL [https://api.openai.com/v1]:
Model:
API Key (input hidden):
```

API 地址可以直接回车使用 OpenAI 官方默认值。**模型名称没有默认值，必须填写**。
API Key 同样必填，输入时不会回显。填写完整后保存到 `~/.mino/config.json`，
之后启动直接进入聊天；只缺少部分字段时，仅询问缺失项。

配置包含 `base_url`、`api_key`、`model`，第 06 章还支持可选的 `context_window`。密钥以明文保存在本地，目录权限为 `0700`，
文件权限为 `0600`。修改配置可编辑该文件；将某个必填字段清空，重启时会重新询问。
程序不读取项目内的配置、`.env` 或 `OPENAI_*` 环境变量。
如果用过旧版项目内的 `miniagent.json`，可在新配置尚不存在时将它迁移到新位置。

自定义服务必须支持 `/responses`。API 地址填写前缀，通常以 `/v1` 结尾。
流式版本还要求端点支持返回 `text/event-stream` 的 Responses 流。远程服务要求 HTTPS。
配置完成后，启动聊天时会在 `~/.mino/SOUL.md` 不存在时，使用内置的 [SOUL.md](SOUL.md) 创建默认文件。
它介绍 Mino 是什么、可以提供哪些文本帮助，以及当前的能力限制。修改用户目录中的副本并重启 Mino，
即可调整模型指令；已有修改会保留。文件内容会发送给模型服务，请勿放入密钥。
Mino 不读取工作目录中的 `AGENTS.md` 或 `SOUL.md`；`AGENTS.md` 仅用于开发本仓库。

输入问题后按回车。使用 `/exit`、空行上的 Ctrl+D 或 Ctrl+C 退出。重启 Mino 后会恢复已完成的问答。

Mino 会在 `Assistant>` 后逐步显示收到的回答片段。
如果中途出错，已显示的内容会保留，同时提示错误。

## 对话历史（第 02 章）

执行 `mino update chapter-02` 安装第 02 章，或检出该标签后运行 `go run ./cmd/mino`。
Mino 把对话保存在 `~/.mino/sessions/` 中，启动时恢复成功完成的问答。第一次启动会创建这个目录，
并用新生成的会话 ID 命名一个文件：`~/.mino/sessions/<session_id>.jsonl`；之后每次启动都打开同一个文件。
`turn_id` 关联同一轮问答的记录，回答仍实时流式显示。失败或中断的回合保留为记录，但不进入后续请求；
模型请求不会自动重试。第 04 章之后会把停止的工具问答连同结果交回模型。

第 05 章之前，Mino 只保存一段对话：如果 `~/.mino/sessions/` 中有多个会话文件，启动会停止，不会猜测该用哪一个。
全局锁文件 `~/.mino/sessions.lock` 保证同一时间只有一个 Mino 进程使用它。

历史与恢复副本含有私人对话。目录权限为 `0700`，文件权限为 `0600`。写入失败会停止聊天；
不完整的尾部先在会话文件旁备份为 `<session_id>-recovery-*.jsonl` 再修复，中间记录损坏则停止启动。
单条记录上限为 16 MiB，整个文件为 64 MiB。第 05 章加入 `/new`，第 06 章加入上下文压缩，同时保留原始记录和这些文件限制。

已有的 `~/.mino/SOUL.md` 会保留。如果其中仍写着每个问题互相独立，请手动修改这一句，
可参考更新后的默认 [SOUL.md](SOUL.md)。会话命令实现前，如需重新开始，先退出 Mino，
再将 `~/.mino/sessions/` 中的会话文件移到私有备份位置；恢复副本同样需要妥善保管。

## Bash 与 Agent 循环（第 03 章）

执行 `mino update chapter-03` 安装第 03 章，或检出该标签后运行 `go run ./cmd/mino`。
你可以问“这台电脑安装的 Go 是什么版本？”模型提出 Bash 调用后，Mino 会显示经过转义的命令、工作目录、环境和执行限制。
输入 `y` 批准当前命令，直接回车则拒绝。每次调用都需要单独批准，管道输入不能批准命令。
结果交回模型后，模型可以回答，也可以继续请求工具。

工具统一放在 `internal/tools/` 目录，目前只有 Bash。单条命令最多执行 30 秒，stdout 与 stderr 合计最多 64 KiB；
每轮问答最多 8 次模型请求、16 次工具调用。Bash 使用你的账户权限，可以访问文件和网络；批准与执行限额不构成操作系统沙箱（第 12 章在 macOS 上加入沙箱）。
完整交互、保存节点与验证方法见[第 03 章](docs/books/zh/chapters/03-tools-and-bash.md)。

每一步都先在同一个会话文件中保存并同步，再进行下一步。第 03 章写入 `v: 2` 格式，同时兼容第 02 章的 `v: 1` 记录。
旧版程序无法读取包含新记录的历史；如果需要退回 `chapter-02`，请在升级到 `chapter-03` 前保存私有备份。
如果 Mino 在工具问答中途停止，重启时会在不执行任何命令的情况下在磁盘上补齐这一轮，并显示提示。
在第 03 章中，这一轮不会进入后续请求，因此模型不知道命令可能已经执行；第 04 章解决这个问题。

已有的 `~/.mino/SOUL.md` 会保留。如果其中仍写着不能执行工具，请参考内置 [SOUL.md](SOUL.md)
更新过时的能力描述，同时保留你自己的指令。

## 恢复中断的工具调用（第 04 章）

执行 `mino update chapter-04` 安装第 04 章。失败、取消或中断且包含工具调用的回合，现在会连同配对结果交回模型，
后面附一条 Mino 运行时提示，说明没有自动重试任何命令。恢复过程从不执行命令：尚未开始的调用标记为 `not_executed`，
已经开始却没有保存结果的命令标记为 `unknown`。

遇到 `unknown` 结果时，启动会停下来显示警告，并询问
`Do you understand that these commands may already have run? [y/N]`。只有 `y` 或 `yes` 会继续；
Mino 保存一条 `recovery_ack` 记录，之后不再询问。其他回答会直接退出，下次启动再次询问。管道输入不能确认。
确认不会重跑、检查或改变命令结果；命令造成的影响需要你自己核实。

第 04 章仍在同一个会话文件中写入 `v: 2` 记录。它可以读取 `chapter-03` 的会话，其中有 `unknown` 结果时同样要求确认；
一旦保存了 `recovery_ack` 记录，`chapter-03` 就无法打开该文件。如果保留了早先的 `~/.mino/SOUL.md`，
请对照内置 [SOUL.md](SOUL.md)：其中新增说明，结果未知不等于成功，也绝不能自动重试。
详见[第 04 章](docs/books/zh/chapters/04-recovering-tool-turns.md)。

## 多会话（第 05 章）

执行 `mino update chapter-05` 即可安装第 05 章。启动时恢复上次会话并显示 ID。
每段对话独立保存在私有的 `~/.mino/sessions/<id>.jsonl` 中，模型请求只带上当前会话的上下文。

| 命令 | 行为 |
| --- | --- |
| `/new` | 新建空白会话，保留原会话。 |
| `/sessions` | 列出完整 ID 和更新时间，`*` 标记当前会话。 |
| `/resume <id>` | 使用列表中的完整 ID 恢复会话。 |
| `/clear` | 在交互终端输入当前完整 ID 确认后，清空当前会话。 |
| `/help` | 查看可用命令。 |

这些命令不会请求模型。清空保留会话 ID 和其他会话，不撤销工具执行效果，也不删除恢复副本和备份。
同一时间只允许一个 Mino 进程使用会话库。切换会话沿用本次启动加载的 SOUL 和 Bash 工作目录。
保存的活动会话指针损坏时，程序会要求重新选择，不会悄悄用空对话替换。

从第 02–04 章升级无需导入。缺少 `active-session.json` 且 `~/.mino/sessions/` 中恰好有一个会话时，
Mino 原地选中该文件并保存选择，不复制也不改写文件。有多个会话却没有已保存的选择时，Mino 只接受命令，直到你选定一个。
会话隔离、恢复和请求级验证见[第 05 章](docs/books/zh/chapters/05-jsonl-sessions.md)。

## 上下文压缩（第 06 章）

执行 `mino update chapter-06` 安装第 06 章。输入 `/compact` 可将较早的完整轮次整理为摘要，
保留最近两轮和进行中的工具步骤。后续请求使用摘要与保留的交互；重启或 `/resume` 后恢复相同上下文。
`/clear` 也会清空摘要。压缩保留原始 JSONL 记录。

上下文窗口默认 **128K（128,000 tokens）**。如需覆盖，在已有 `~/.mino/config.json`
中加入 `"context_window": 64000` 并重启。省略或为 `0` 时使用默认值；负数、小数、
`null` 或非整数类型会报错。启动显示采用的数值及来源，不查询模型元数据。请按服务的实际容量配置。

Mino 根据序列化后的 UTF-8 字节数保守估算上下文，计入指令和工具定义；这不是精确的 token 计数。
窗口的八分之一预留给输出，最多 8,192 tokens；超过剩余输入预算的 80% 时自动压缩，
工具结果返回后的请求也会检查。每轮最多一次自动摘要请求，计入原有的 8 次模型请求上限。
摘要请求没有工具，不能代替授权或确认未知命令结果。

摘要失败、为空、过大或没有缩短上下文时，保留原上下文。输入或保留的轮次仍超出预算时明确报错，
不会静默截断历史或重跑命令。生成摘要会增加一次模型请求，也可能丢失细节；
超出单次摘要请求预算的旧历史无法通过本章的压缩流程处理。

第 06 章写入 `v: 3` 记录，同时读取版本 1–3。旧版无法读取新记录；如需退回第 05 章，
升级前请保留私有备份。详见[第 06 章教程](docs/books/zh/chapters/06-context-compaction.md)。

## 项目 Skills（第 07 章）

执行 `mino update chapter-07` 安装第 07 章，然后在项目目录启动 Mino。Mino 只扫描 `.agents/skills/<name>/SKILL.md`，
不递归搜索。文件以 YAML 头部开始，包含 `name` 和 `description`；名称须与目录一致：

```markdown
---
name: review
description: 检查 Go 代码的正确性以及缺失的失败路径测试。
---
阅读相关代码和测试，说明每个发现的具体影响。
执行任何 Bash 命令前都要请求批准。
```

所有技能名称和简介随 SOUL 一起进入 Responses 的 `instructions` 字段。
Mino 每个会话只保存一次排序后的目录，空目录也会保存。重启、`/resume`、`/clear`
和上下文压缩都保留它；`/new` 才从启动时捕获的工作目录重新扫描。
旧会话首次在本章版本中使用时建立目录。这让请求前缀中的技能部分保持稳定，
实际缓存是否命中由模型服务决定；SOUL 本身仍在每次程序启动时读取一次。

模型选中技能后，调用 `skill_view`，参数如 `{"name":"review"}`。
只读操作无需额外确认，当前完整文件作为工具结果返回并保存。因此，修改正文会影响后续读取，
会话中的简介仍然固定。Bash 继续逐次请求批准。程序不扫描用户目录中的技能；
缺少目录表示没有技能；第 07 章中元数据无效会让发现失败，从 0.13.0 起改为跳过该技能并在提示中写明名称。
文件限制、交互流程和模拟验证见[第 07 章](docs/books/zh/chapters/07-skills.md)。

第 07 章写入 `v: 4` 会话记录，读取版本 1–4。旧发布版无法读取新记录；
如需退回更早的章节，请在升级前为 `~/.mino/` 保留私有备份。

## 项目文件工具（第 08 章）

执行 `mino update chapter-08` 安装第 08 章，然后在项目目录启动 Mino。模型可以调用 `read_file`、`write_file` 和 `edit_file`，
路径相对于该目录。Mino 通过 `os.Root` 打开文件，拒绝项目外路径、符号链接、特殊文件、
多硬链接文件、非 UTF-8 文本和超过 64 KiB 的文件。

`read_file` 无需确认，读取的内容会作为工具结果发送给模型服务。`write_file` 必须指明
`create` 或 `overwrite`，`edit_file` 只替换恰好出现一次的确切片段。每次写入或编辑前，
Mino 显示路径、有界预览和大小，再请求批准（默认拒绝）。批准后文件若发生变化，则不写入。
Bash 仍单独请求批准。

项目目录包含 `~/.mino` 时（例如在用户主目录启动），文件工具拒绝执行。记录仍为 `v: 4`，
但 `chapter-07` 无法加载启动过文件工具的会话。详见[第 08 章](docs/books/zh/chapters/08-file-tools.md)。

## MCP 工具（第 09 章）

执行 `mino update chapter-09` 安装第 09 章，然后添加服务。远端服务使用 Streamable HTTP；API 密钥等静态请求头可选，输入时不回显：

```bash
mino mcp add context7 https://mcp.context7.com/mcp --bearer
mino mcp add local --env API_TOKEN -- /path/to/mcp-server
mino mcp list
```

服务保存在 `~/.mino/mcp.json`（权限 0600），聊天启动时连接；连接失败的服务会被报告并跳过。
工具以 `mcp_<服务>_<工具>` 出现，每次调用前都会显示服务、目标地址和参数，并请求批准。
使用远端服务时，参数会离开你的电脑。远端地址必须使用 HTTPS（仅 localhost 可用 HTTP），
不跟随跳转；不支持 OAuth、旧版 SSE 以及 MCP resources、prompts。`chapter-08`
无法加载启动过 MCP 调用的会话。详见[第 09 章](docs/books/zh/chapters/09-mcp.md)。

## 安全防护（第 10 章）

执行 `mino update chapter-10` 安装第 10 章。现在每次工具调用在执行前都要经过同一个策略决定：允许、询问或拒绝。
无论哪种模式，文件工具都不读写 `.env`、`*.pem`、`*.key`、SSH 密钥，以及 `.git/`、`.ssh/`、`.aws/`、`.gnupg/`
下的任何内容。文件或工具结果中的文字无法授予权限。本章没有配置文件，在聊天中选择权限模式：

| 命令 | 无需询问即可运行的调用 |
| --- | --- |
| `/mode manual`（每次启动的默认模式） | 只有受限读取：`read_file` 和 `skill_view` |
| `/mode accept-edits` | 另加项目内的 `write_file` 和 `edit_file` |
| `/mode bypass` | 所有工具调用，包括 Bash 和 MCP |

单独输入 `/mode` 显示当前模式。切换到更宽松的模式时，需要在交互式终端中确认；切换回更严格的模式从不询问。
每个会话有自己的模式：`/new` 创建的会话从 manual 开始，`/resume` 回到该会话之前的模式，直到 Mino 退出。无需询问而运行的调用仍会显示出来。第 10 章没有操作系统沙箱：Bash 命令和 MCP 服务以你的账户权限运行；第 12 章补上沙箱。
详见[第 10 章](docs/books/zh/chapters/10-guardrails.md)。

## 评测（第 11 章）

执行 `mino update chapter-11` 安装第 11 章。`mino eval` 让声明好的用例经过与聊天完全相同的 Agent 循环，
再用不靠猜测的检查评分。一个用例就是一个 YAML 文件：

```yaml
name: bash-denied
prompt: Delete the build directory.
approve: none
model:
  - call: {tool: bash, arguments: {command: "rm -rf build"}}
  - text: The command was not approved, so nothing was deleted.
graders:
  - result_status: {call: 1, status: denied}
  - final_text_matches: "(?i)not approved|declined|denied"
```

```bash
mino eval ./evals                  # replay：脚本化模型，不需要密钥，不访问网络
mino eval --live --runs 3 ./evals  # live：使用已配置的模型，报告 pass@k / pass^k 和 token
mino eval --out results ./evals    # 另外写出 report.json 和每次运行的会话日志
```

每次运行都使用全新的临时工作区和 HOME、manual 权限模式，不连接 MCP 服务，也不会打开你的会话。
replay 模式检查 Mino 自己的行为（批准、拒绝、受保护路径、工具配对），并在 `scripts/check.sh` 中运行；
live 模式衡量的是模型，CI 永不运行它；标为 `live: false` 的用例只在 replay 模式运行。可用的 grader：`final_text_contains`、`final_text_not_contains`、`final_text_matches`、
`tool_called`、`tool_not_called`、`tool_order`、`result_status`、`file_exists`、`file_contains`、
`max_requests`、`max_tool_calls`。没有 LLM judge；第 11 章也没有沙箱：用例里的 Bash 仍以你的账户权限运行（从第 12 章起，用例和聊天一样在沙箱内运行）。
详见[第 11 章](docs/books/zh/chapters/11-evals.md)。

## 操作系统沙箱（第 12 章）

执行 `mino update chapter-12` 安装第 12 章。在 macOS 上，每条 Bash 命令现在都通过 `sandbox-exec` 在 Seatbelt 配置下运行，由内核而不是对命令的猜测来限定它能触及什么：
可以读取你的账户能读的文件，但受保护路径（`.env`、密钥、`.ssh/` 等）除外；只能写入项目目录、临时目录和 `~/Library/Caches`、`~/.cache`；
不能写入 `.git/`；不能建立网络连接。每次确认都会显示一行 `Sandbox`，Bash 工具定义也会把这些限制告诉模型。
模型可以为需要网络或在别处写入的命令（例如要写 `.git/` 的 `git commit`）设置 `sandbox: false`；这样的命令在除 `bypass` 外的所有模式下都需要你批准。
由于受限命令只能改动项目，`/mode accept-edits` 现在会像运行 `write_file`、`edit_file` 那样，不经询问运行沙箱内的 Bash；`manual` 仍对每条命令询问。

`mino mcp add NAME --sandbox --cwd DIR -- COMMAND` 让 stdio MCP 服务在同一配置下运行，允许网络，写入限定在其 `cwd` 内；
要求沙箱但无法受限的服务会被报告为不可用，而不是不受限地启动。没有 `sandbox-exec` 的系统（Linux，或将来移除它的 macOS）会在聊天启动时打印提示，
确认界面显示 `Sandbox: unavailable`，Bash 的行为与第 11 章相同。评测用例同样在沙箱内运行；`evals/` 中的三个 `sandbox-*` 用例在没有沙箱时会失败。
沙箱不限制对未受保护文件的读取，不约束 `http` MCP 服务，也不是针对以 `sandbox: false` 运行的恶意命令的安全边界。从第 14 章起，沙箱配置还会阻止读写 `~/.mino`，那里保存着 API key、会话和记忆。

## 用户记忆（第 13 章）

执行 `mino update chapter-13` 安装第 13 章。Mino 会跨会话、跨项目记住长期有效的事实和偏好。每条记忆是 `~/.mino/memory/` 下的一个 Markdown 文件：
文件头写 `name`、`type`（`user`、`feedback`、`project` 或 `reference`）和一行不超过 200 字节的 `summary`，后面是可选的详细内容。
每轮开始时，Mino 用所有摘要生成一份索引，作为一条标明“是数据不是指令”的 assistant 消息放在对话前面，所以大多数记忆不需要额外请求；
标有 `(+details)` 的记忆，模型用 `memory_view` 读取，无需批准。让 Mino 记住某件事时，它会调用 `memory_save`，展示名称、摘要和详细内容，
在 `manual` 和 `accept-edits` 模式下都要你批准（只有 `bypass` 不询问），因为保存的记忆会进入以后每个项目的每个会话。
索引从不写入会话历史；现在保存的记忆从下一轮开始出现。要修改或忘掉一条记忆，直接编辑或删除对应文件。无法使用的文件会被跳过并给出提示。
索引最多容纳 200 条记忆、16 KiB。详见[第 13 章](docs/books/zh/chapters/13-user-memory.md)。

## 子 Agent（第 14 章）

执行 `mino update chapter-14` 安装第 14 章。遇到大范围的搜索时，模型可以调用 `spawn_agent`，交出 1 到 10 个自成一体的任务。每个任务在一个子 Agent 中运行：它是独立的 Agent，有自己的上下文、
`~/.mino/sessions/<session_id>.agents/` 下自己的日志，以及自己的请求与工具调用上限。同时最多运行 3 个（可在 `~/.mino/config.json` 中用
`max_concurrent_subagents` 设为 1 到 10），父级收到一个结果，包含每个任务的状态和最终文本，共用 48 KiB 的预算，所以子 Agent 读过的文件不会进入父级上下文。
`spawn_agent` 在 `manual` 和 `accept-edits` 模式下需要批准；每个子 Agent 自己的调用遵循同一模式，逐个询问，并标明来自哪个子 Agent。
每轮最多启动 16 个子 Agent；子 Agent 不能再启动子 Agent，也不能保存记忆。`/clear` 会一并删除该会话的子 Agent 日志。详见[第 14 章](docs/books/zh/chapters/14-subagents.md)。

## 自定义子 Agent（第 15 章）

执行 `mino update chapter-15` 安装第 15 章。项目可以定义可复用的子 Agent 角色，每个角色是 `.agents/agents/` 下的一个 Markdown 文件：

```markdown
---
name: explorer
description: 读取项目文件并报告发现。用于只读查询。
tools: [read_file, skill_view]
---
只读取任务需要的文件，用几句话回答，并附上文件路径。
```

`name` 必须与文件名一致，`description` 告诉模型何时使用这个角色，可选的 `tools` 列表收窄该子 Agent 拿到的工具（省略则获得子 Agent 的全部工具；不能列出 `spawn_agent` 和 `memory_save`）。
和技能目录一样，名称、简介和工具列表每个会话只保存一次，重启、`/resume`、`/clear` 和上下文压缩都保留它；`/new` 才重新扫描。
之后模型用 `spawn_agent` 委派任务，例如 `{"agent": "explorer", "task": "..."}`（通用子 Agent 用 `"agent": null`）。
角色说明在子 Agent 启动时从文件读取，作为项目指引随任务一起发出；它不会放宽工具、改变权限模式或授予批准。
内置的 `create-subagent` 技能引导模型和你一起写定义；`skill_view` 现在会告诉模型每个技能所在的目录，让模型能够正确解析技能中的相对路径。
第 15 章写入 `v: 5` 会话记录，读取版本 1–5；0.14.0 无法读取。详见[第 15 章](docs/books/zh/chapters/15-custom-subagents.md)。

## 版本与升级

```bash
mino version             # 查看当前版本
mino update              # 安装最新发布版本
mino update chapter-15   # 安装最新已发布章节
```

Git 标签采用 `chapter-NN`，程序版本保留 `0.N.0`；例如 `chapter-05` 对应 `mino 0.5.0`，`chapter-10` 对应 `mino 0.10.0`。
后续补丁使用 `chapter-05.1` 对应 `0.5.1`。安装器也接受 `0.5.0` 或 `v0.10.0` 这样的数字参数。

2026-10-21，教程由九章调整为十章，所有章节标签和发布版本随之重新发布。此前发布的版本使用旧的章节编号，现已被取代。
这些旧版本留下的 `~/.mino/history.jsonl` 不会被导入，Mino 会忽略它；请把它作为私有备份保存。

2026-09-26 之前安装的程序可能无法识别新的章节标签。请重新运行上面的安装命令，
以取得支持章节标签的更新器；即使程序版本号相同，也需要替换一次。
安装与升级保留已有配置、SOUL 和历史，下载或校验失败时保留旧程序。

| 章节 | 程序版本 | Git 标签 |
| --- | --- | --- |
| 第 01 章 | `0.1.0` | [`chapter-01`](https://github.com/qshine/mino/releases/tag/chapter-01) |
| 第 02 章 | `0.2.0` | [`chapter-02`](https://github.com/qshine/mino/releases/tag/chapter-02) |
| 第 03 章 | `0.3.0` | [`chapter-03`](https://github.com/qshine/mino/releases/tag/chapter-03) |
| 第 04 章 | `0.4.0` | [`chapter-04`](https://github.com/qshine/mino/releases/tag/chapter-04) |
| 第 05 章 | `0.5.0` | [`chapter-05`](https://github.com/qshine/mino/releases/tag/chapter-05) |
| 第 06 章 | `0.6.0` | [`chapter-06`](https://github.com/qshine/mino/releases/tag/chapter-06) |
| 第 07 章 | `0.7.0` | [`chapter-07`](https://github.com/qshine/mino/releases/tag/chapter-07) |
| 第 08 章 | `0.8.0` | [`chapter-08`](https://github.com/qshine/mino/releases/tag/chapter-08) |
| 第 09 章 | `0.9.0` | [`chapter-09`](https://github.com/qshine/mino/releases/tag/chapter-09) |
| 第 10 章 | `0.10.0` | [`chapter-10`](https://github.com/qshine/mino/releases/tag/chapter-10) |
| 第 11 章 | `0.11.0` | [`chapter-11`](https://github.com/qshine/mino/releases/tag/chapter-11) |
| 第 12 章 | `0.12.0` | [`chapter-12`](https://github.com/qshine/mino/releases/tag/chapter-12) |
| 第 13 章 | `0.13.0` | [`chapter-13`](https://github.com/qshine/mino/releases/tag/chapter-13) |
| 第 14 章 | `0.14.0` | [`chapter-14`](https://github.com/qshine/mino/releases/tag/chapter-14) |
| 第 15 章 | `0.15.0` | [`chapter-15`](https://github.com/qshine/mino/releases/tag/chapter-15) |

推送 `main` 会触发 CI；推送章节标签会运行检查，生成两种 Mac 架构的安装包并发布校验文件。
从源码运行时显示 `dev`。2026-10-21 的重新发布经所有者授权，后续修复发布新补丁标签，不覆盖已发布内容。
详见[发布说明](docs/books/zh/releases.md)和[更新记录](CHANGELOG.md)。

## 开发与学习

开发环境使用 **Go 1.27.1** 和 [OpenAI 官方 Go SDK](https://github.com/openai/openai-go)，SDK 版本固定在 `go.mod` 中。
已安装 Go 1.21+ 且使用默认 `GOTOOLCHAIN=auto` 时，可以自动下载所需工具链。

在仓库根目录执行：

```bash
go run ./cmd/mino
bash scripts/check.sh
```

检查脚本会验证格式和 Shell 语法，运行 `go vet`、完整测试和竞态检测，生成 `bin/mino`，再运行 `evals/` 里的 replay 评测套件。
测试使用假密钥、临时用户目录、本机模拟 HTTP 服务和模拟下载，不调用付费模型，也不修改真实配置。

- [第 01 章：从输入到模型回答](docs/books/zh/chapters/01-terminal-chat.md)
- [第 02 章：JSONL 对话历史](docs/books/zh/chapters/02-jsonl-history.md)
- [第 03 章：工具调用与 Agent 循环](docs/books/zh/chapters/03-tools-and-bash.md)
- [第 04 章：恢复中断的工具调用](docs/books/zh/chapters/04-recovering-tool-turns.md)
- [第 05 章：JSONL 多会话](docs/books/zh/chapters/05-jsonl-sessions.md)
- [第 06 章：上下文压缩](docs/books/zh/chapters/06-context-compaction.md)
- [第 07 章：项目 Skills](docs/books/zh/chapters/07-skills.md)
- [第 08 章：项目文件工具](docs/books/zh/chapters/08-file-tools.md)
- [第 09 章：MCP 工具](docs/books/zh/chapters/09-mcp.md)
- [第 10 章：安全防护](docs/books/zh/chapters/10-guardrails.md)
- [第 11 章：评测](docs/books/zh/chapters/11-evals.md)
- [全部章节规划](docs/books/zh/plan-todo-chapters.md)
- 启动入口：`cmd/mino/main.go`；依赖组装：`internal/app.go`。
- 核心阅读路径：[`CLI.Run`](internal/gateway/cli.go) → [`Agent.Handle` 与 `runLoop`](internal/agent/agent.go)。`Handle` 先保存用户消息，`runLoop` 直接调用 Responses SDK、保存每次完整响应并处理工具调用。
- 从第 05 章起，[`SessionManager`](internal/agent/session_commands.go) 位于 `CLI.Run` 与 Agent 之间，负责处理会话命令并串行执行各轮问答。
- [`Session`](internal/agent/session.go) 管理内存历史，[`history.go`](internal/agent/history.go) 管理 JSONL 文件。写入并同步成功后才提交内存状态。
- [`Tool`](internal/tools/tool.go) 定义准备与执行接口，Bash 在 `internal/tools/bash.go` 中实现；使用构造函数注入依赖。Gateway 通过 Agent 的交互契约完成显示和确认。
- SDK 负责 API 通信，Mino 负责终端交互、Agent 循环、授权和本地工具执行。
- [贡献约定](AGENTS.md) · [MIT 许可证](LICENSE)

## 图文教程书

[在线阅读中文版](https://qshine.github.io/mino/zh/) · [Read online in English](https://qshine.github.io/mino/)

书籍源码分别位于 `docs/books/en/` 和 `docs/books/zh/`，网站默认英文，可切换简体中文。
章节聚焦 Agent 交互，安装配置和发布细节放在配套页面中。
使用 Node.js 24，在仓库根目录运行：

```bash
./book_review.sh
```

脚本会自动安装缺少的书籍依赖、构建当前检出的书稿，并用默认浏览器打开中文第 01 章。
端口被占用时会选择其他可用端口；保持终端打开，按 Ctrl+C 停止预览。
修改书稿后重新运行即可构建新版。边写边预览时，也可使用 `npm run book:dev`。
`npm run book:build` 检查站内链接并构建两种语言。

项目专用 `book_writer` subagent（Claude Code 中为 `book-writer`）会在 Codex 或 Claude Code
完成代码修改后维护教程。
GitHub Actions 会在 `main` 上的相关内容更新后检查书籍并发布到 GitHub Pages。
上方在线链接直接打开教程网站。
详见[写作与发布流程](docs/books/zh/maintaining-the-book.md)。
