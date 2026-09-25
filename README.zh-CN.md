# Mino

[English](README.md) | 简体中文

**用 Go 从零构建自己的 Agent：从一次交互理解原理，用代码和实验验证能力。**

Mino 是面向 Agent 入门者的分章教程，使用 Go 和 OpenAI Responses API。
每章从一段可以观察的交互开始，看清模型收到什么、程序负责什么，再用小实验检查结果与边界。
阅读源码需要了解 Go 的变量、函数和基本错误处理；第一次接触模型 API 也可以从第 01 章开始。

**在线阅读：**[中文教程](https://qshine.github.io/mino/zh/) · [English book](https://qshine.github.io/mino/)

**当前进度：**本快照包含第 01–04 章，对应发布版本 `chapter-04`（程序版本 `0.4.0`）。
第 01 章实现流式终端问答。
第 02 章加入 JSONL 对话历史和重启恢复。
第 03 章加入逐次批准的 Bash 执行与 Agent 循环。
第 04 章如实恢复中断的工具调用：结果交回模型，结果未知的操作需要你确认。
Gateway 与 Agent 目录从第 01 章起保持一致。
后续章节见[章节规划](docs/books/zh/plan-todo-chapters.md)。

**作者：qqling | AI Builder。**我想从零构建一个属于自己的 Agent，把持续研究和实践中的理解整理成入门教程。
你可以在[作者介绍](docs/books/zh/about-author.md)中了解我的创作初衷，并通过 X、GitHub 或小红书关注和交流。

## 安装

支持 **macOS 13 及以上版本**，提供 **Apple Silicon 和 Intel Mac** 安装包。
仓库和安装包已公开，无需安装 Go、GitHub CLI，也无需登录 GitHub。
安装程序使用 macOS 自带的 `curl` 下载。

在 Bash 或 zsh 中执行这一行命令：

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-04 && export PATH="$HOME/.mino/bin:$PATH"
```

安装程序会识别 Mac 架构，下载本快照对应的 `chapter-04` 版本，校验 SHA-256 和程序版本，然后安装到 `~/.mino/bin/mino`。
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

配置包含 `base_url`、`api_key`、`model`。密钥以明文保存在本地，目录权限为 `0700`，
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
每轮问答最多 8 次模型请求、16 次工具调用。Bash 使用你的账户权限，可以访问文件和网络；批准与执行限额不构成操作系统沙箱。
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

## 版本与升级

```bash
mino version             # 查看当前版本
mino update              # 安装最新发布版本
mino update chapter-04   # 安装本快照对应的章节
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

检查脚本会验证格式和 Shell 语法，运行 `go vet`、完整测试和竞态检测，并生成 `bin/mino`。
测试使用假密钥、临时用户目录、本机模拟 HTTP 服务和模拟下载，不调用付费模型，也不修改真实配置。

- [第 01 章：从输入到模型回答](docs/books/zh/chapters/01-terminal-chat.md)
- [第 02 章：JSONL 对话历史](docs/books/zh/chapters/02-jsonl-history.md)
- [第 03 章：工具调用与 Agent 循环](docs/books/zh/chapters/03-tools-and-bash.md)
- [第 04 章：恢复中断的工具调用](docs/books/zh/chapters/04-recovering-tool-turns.md)
- [全部章节规划](docs/books/zh/plan-todo-chapters.md)
- 启动入口：`cmd/mino/main.go`；依赖组装：`internal/app.go`。
- 核心阅读路径：[`CLI.Run`](internal/gateway/cli.go) → [`Agent.Handle` 与 `runLoop`](internal/agent/agent.go)。`Handle` 先保存用户消息，`runLoop` 直接调用 Responses SDK、保存每次完整响应并处理工具调用。
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

项目专用 `book_writer` subagent 会在 Codex 完成代码修改后维护教程。
GitHub Actions 会在 `main` 上的相关内容更新后检查书籍并发布到 GitHub Pages。
上方在线链接直接打开教程网站。
详见[写作与发布流程](docs/books/zh/maintaining-the-book.md)。
