# Mino

[English](README.md) | 简体中文

**用 Go 从零构建自己的 Agent：从一次交互理解原理，用代码和实验验证能力。**

Mino 是面向 Agent 入门者的分章教程，使用 Go 和 OpenAI Responses API。
每章从一段可以观察的交互开始，看清模型收到什么、程序负责什么，再用小实验检查结果与边界。
阅读源码需要了解 Go 的变量、函数和基本错误处理；第一次接触模型 API 也可以从第一章开始。

**在线阅读：**[中文教程](https://qshine.github.io/mino/zh/) · [English book](https://qshine.github.io/mino/)

**当前进度：**本快照包含已发布的第 01–05 章，标签为 `chapter-01` 至 `chapter-05`，程序版本对应 `0.1.0` 至 `0.5.0`。第一章实现流式终端问答，第二章加入 JSONL 历史和重启恢复。第三章加入逐次批准的 Bash 执行与 Agent 循环。第四章加入独立会话、新建、恢复和确认清空。Gateway 与 Agent 目录从第一章起保持一致。详见[章节规划](docs/books/zh/plan-todo-chapters.md)。

第五章加入上下文压缩：手动 `/compact`、自动预算检查和摘要持久化。

**作者：qqling | AI Builder。**我想从零构建一个属于自己的 Agent，把持续研究和实践中的理解整理成入门教程。
你可以在[作者介绍](docs/books/zh/about-author.md)中了解我的创作初衷，并通过 X、GitHub 或小红书关注和交流。

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

配置包含 `base_url`、`api_key`、`model`，第五章还支持可选的 `context_window`。密钥以明文保存在本地，目录权限为 `0700`，
文件权限为 `0600`。修改配置可编辑该文件；将某个必填字段清空，重启时会重新询问。
程序不读取项目内的配置、`.env` 或 `OPENAI_*` 环境变量。
如果用过旧版项目内的 `miniagent.json`，可在新配置尚不存在时将它迁移到新位置。

自定义服务必须支持 `/responses`。API 地址填写前缀，通常以 `/v1` 结尾。
流式版本还要求端点支持返回 `text/event-stream` 的 Responses 流。远程服务要求 HTTPS。
配置完成后，启动聊天时会在 `~/.mino/SOUL.md` 不存在时，使用内置的 [SOUL.md](SOUL.md) 创建默认文件。
它介绍 Mino 是什么、可以提供哪些文本帮助，以及当前的能力限制。修改用户目录中的副本并重启 Mino，
即可调整模型指令；已有修改会保留。文件内容会发送给模型服务，请勿放入密钥。
Mino 不读取工作目录中的 `AGENTS.md` 或 `SOUL.md`；`AGENTS.md` 仅用于开发本仓库。
重新发布的 `chapter-01` 已包含这个身份文件。

输入问题后按回车。使用 `/exit`、空行上的 Ctrl+D 或 Ctrl+C 退出。重启 Mino 后会恢复已完成的问答。

Mino 会在 `Assistant>` 后逐步显示收到的回答片段。
如果中途出错，已显示的内容会保留，同时提示错误。

## 对话历史（第二章）

`chapter-02` 引入第二章的历史保存；第三章加入下文的工具执行恢复。
第二、三章将记录追加到 `~/.mino/history.jsonl`，启动时恢复成功完成的问答。
一个本地文件保存一段会话，`turn_id` 关联同一轮问答的记录，回答仍实时流式显示。
在 `chapter-02` 中，失败或中断的回合保留为记录，但不进入后续请求。第三章还会恢复已配对的工具结果，
因为停止前命令可能已经产生影响。模型请求和命令都不会自动重试。

历史与恢复副本含有私人对话，文件权限为 `0600`，同一时间只允许一个 Mino 进程使用历史。
写入失败会停止聊天；不完整的尾部先备份再修复，中间记录损坏则停止启动。
单条记录上限为 16 MiB，整个文件为 64 MiB。第四章加入 `/new`，第五章加入上下文压缩，同时保留原始记录和这些文件限制。

已有的 `~/.mino/SOUL.md` 会保留。如果其中仍写着每个问题互相独立，请手动修改这一句，
可参考更新后的默认 [SOUL.md](SOUL.md)。会话命令实现前，如需重新开始，先退出 Mino，
再将 `history.jsonl` 移到私有备份位置；恢复副本同样需要妥善保管。

## Bash 与 Agent 循环（第三章）

执行 `mino update chapter-03` 安装第三章，或检出该标签后运行 `go run ./cmd/mino`。
你可以问“这台电脑安装的 Go 是什么版本？”模型提出 Bash 调用后，Mino 会显示经过转义的命令、工作目录、环境和执行限制。
输入 `y` 批准当前命令，直接回车则拒绝。每次调用都需要单独批准，管道输入不能批准命令。
结果交回模型后，模型可以回答，也可以继续请求工具。

工具统一放在 `internal/tools/` 目录，目前只有 Bash。单条命令最多执行 30 秒，stdout 与 stderr 合计最多 64 KiB；
每轮问答最多 8 次模型请求、16 次工具调用。Bash 使用你的账户权限，可以访问文件和网络；批准与执行限额不构成操作系统沙箱。
完整交互、恢复行为与验证方法见[第三章](docs/books/zh/chapters/03-tools-and-bash.md)。

第三、四章的历史文件写入 `v: 2` 格式，同时兼容第二章的 `v: 1` 记录。旧版程序无法读取包含新记录的历史；
如果需要退回 `chapter-02`，请在升级到 `chapter-03` 前保存私有备份。
命令开始后没有保存结果便中断，会标记为 `unknown`。继续聊天前，你需要确认理解“操作可能已经发生”；
Mino 不会在恢复时重新执行它。

已有的 `~/.mino/SOUL.md` 会保留。如果其中仍写着不能执行工具，请参考内置 [SOUL.md](SOUL.md)
更新过时的能力描述，同时保留你自己的指令。

## 多会话（第四章）

执行 `mino update chapter-04` 即可安装第四章。启动时恢复上次会话并显示 ID。
每段对话独立保存在私有的 `~/.mino/sessions/<id>.jsonl` 中，模型请求只带上当前会话的上下文。

| 命令 | 行为 |
| --- | --- |
| `/new` | 新建空白会话，保留原会话。 |
| `/sessions` | 列出完整 ID 和更新时间，`*` 标记当前会话。 |
| `/resume <id>` | 使用列表中的完整 ID 恢复会话。 |
| `/clear` | 在交互终端输入当前完整 ID 确认后，清空当前会话。 |
| `/help` | 查看可用命令。 |

这些命令不会请求模型。清空保留会话 ID 和其他会话，不撤销工具执行效果，也不删除留档和备份。
同一时间只允许一个 Mino 进程使用会话库。切换会话沿用本次启动加载的 SOUL 和 Bash 工作目录。
保存的活动会话指针损坏时，程序会要求重新选择，不会悄悄用空对话替换。

首次升级会在持有旧历史锁时导入 `history.jsonl`，并保留它作为私有留档。
迁移中断后复用原目标，遇到内容冲突不会覆盖。旧版本仍使用旧留档，看不到新会话中的后续内容。
会话隔离、恢复和请求级验证见[第四章](docs/books/zh/chapters/04-jsonl-sessions.md)。

## 上下文压缩（第五章）

执行 `mino update chapter-05` 安装第五章。输入 `/compact` 可将较早的完整轮次整理为摘要，
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

第五章写入 `v: 3` 记录，同时读取版本 1–3。旧版无法读取新记录；如需退回第四章，
升级前请保留私有备份。详见[第五章教程](docs/books/zh/chapters/05-context-compaction.md)。

## 版本与升级

```bash
mino version             # 查看当前版本
mino update              # 安装最新发布版本
mino update chapter-05   # 安装最新已发布章节
```

Git 标签采用 `chapter-NN`，程序版本保留 `0.N.0`；例如 `chapter-04` 对应 `mino 0.4.0`。
后续补丁使用 `chapter-04.1` 对应 `0.4.1`。安装器也接受 `0.4.0` 或 `v0.4.0` 这样的数字参数。

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

推送 `main` 会触发 CI；推送章节标签会运行检查，生成两种 Mac 架构的安装包并发布校验文件。
从源码运行时显示 `dev`。本次标签调整经所有者授权，后续修复发布新补丁标签，不覆盖已发布内容。
详见[发布说明](docs/books/zh/releases.md)和[更新记录](CHANGELOG.md)。

## 开发与学习

开发环境使用 **Go 1.27.1** 和 [OpenAI 官方 Go SDK](https://github.com/openai/openai-go)，SDK 版本固定在 `go.mod` 中。
已安装 Go 1.21+ 且使用默认 `GOTOOLCHAIN=auto` 时，可以自动下载所需工具链。

在仓库根目录执行：

```bash
go run ./cmd/mino
bash scripts/check.sh
```

检查脚本会验证格式和 Shell 语法，运行 `go vet`、完整测试和竞态检测，并生成 `bin/mino`。
测试使用假密钥、临时用户目录、本机模拟 HTTP 服务和模拟下载，不调用付费模型，也不修改真实配置。

- [第一章：从输入到模型回答](docs/books/zh/chapters/01-terminal-chat.md)
- [第二章：JSONL 对话历史](docs/books/zh/chapters/02-jsonl-history.md)
- [第三章：工具调用与 Agent 循环](docs/books/zh/chapters/03-tools-and-bash.md)
- [第四章：JSONL 多会话](docs/books/zh/chapters/04-jsonl-sessions.md)
- [第五章：上下文压缩](docs/books/zh/chapters/05-context-compaction.md)
- [全部章节规划](docs/books/zh/plan-todo-chapters.md)
- 启动入口：`cmd/mino/main.go`；依赖组装：`internal/app.go`。
- 核心阅读路径：[`CLI.Run`](internal/gateway/cli.go) → [`SessionManager`](internal/agent/session_commands.go) → [`Agent.Handle` 与 `runLoop`](internal/agent/agent.go)。`Handle` 先保存用户消息，`runLoop` 直接调用 Responses SDK、保存每次完整响应并处理工具调用。
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

脚本会自动安装缺少的书籍依赖、构建当前检出的书稿，并用默认浏览器打开中文第一章。
端口被占用时会选择其他可用端口；保持终端打开，按 Ctrl+C 停止预览。
修改书稿后重新运行即可构建新版。边写边预览时，也可使用 `npm run book:dev`。
`npm run book:build` 检查站内链接并构建两种语言。

项目专用 `book_writer` subagent 会在 Codex 完成代码修改后维护教程。
GitHub Actions 会在 `main` 上的相关内容更新后检查书籍并发布到 GitHub Pages。
上方在线链接直接打开教程网站。
详见[写作与发布流程](docs/books/zh/maintaining-the-book.md)。
