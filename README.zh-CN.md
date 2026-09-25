# Mino

[English](README.md) | 简体中文

用 Go 从零实现终端 Agent 的分章教程。第一章通过 OpenAI Responses API 实现独立的一问一答。
支持 **macOS 13 及以上版本**，提供 **Apple Silicon 和 Intel Mac** 安装包。

**在线阅读：**[中文教程](https://qshine.github.io/mino/zh/) · [English book](https://qshine.github.io/mino/)

## 安装

仓库和安装包已公开，无需安装 Go、GitHub CLI，也无需登录 GitHub。
安装程序使用 macOS 自带的 `curl` 下载。

在 Bash 或 zsh 中执行这一行命令：

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-01 && export PATH="$HOME/.mino/bin:$PATH"
```

安装程序会识别 Mac 架构，下载本章发布版本，校验 SHA-256 和程序版本，然后安装到 `~/.mino/bin/mino`。
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
文件权限为 `0600`。修改配置可编辑该文件；将某字段清空，重启时会重新询问。
程序不读取项目内的配置、`.env` 或 `OPENAI_*` 环境变量。
如果用过旧版项目内的 `miniagent.json`，可在新配置尚不存在时将它迁移到新位置。

自定义服务必须支持 `/responses`。API 地址填写前缀，通常以 `/v1` 结尾。
流式版本还要求端点支持返回 `text/event-stream` 的 Responses 流。远程服务要求 HTTPS。
配置完成后，启动聊天时会在 `~/.mino/SOUL.md` 不存在时，使用内置的 [SOUL.md](SOUL.md) 创建默认文件。
它介绍 Mino 是什么、可以提供哪些文本帮助，以及当前的能力限制。修改用户目录中的副本并重启 Mino，
即可调整模型指令；已有修改会保留。文件内容会发送给模型服务，请勿放入密钥。
Mino 不读取工作目录中的 `AGENTS.md` 或 `SOUL.md`；`AGENTS.md` 仅用于开发本仓库。
重新发布的 `chapter-01` 已包含这个身份文件。

输入问题后按回车。使用 `/exit`、空行上的 Ctrl+D 或 Ctrl+C 退出。第一章不保留聊天历史。

重新发布的 `chapter-01` 流式版本会在 `Assistant>` 后逐步显示收到的回答片段。
如果中途出错，已显示的内容会保留，同时提示错误。请重新执行上方安装命令获取流式输出，
即便当前版本已经显示 `0.1.0`；更早的同号版本会等待完整回答后一次性显示。

## 版本与升级

```bash
mino version             # 查看当前版本
mino update              # 安装最新发布版本
mino update chapter-01   # 安装本快照对应章节
```

Git 标签采用 `chapter-NN`，程序版本保留 `0.N.0`；例如 `chapter-04` 对应 `mino 0.4.0`。
后续补丁使用 `chapter-04.1` 对应 `0.4.1`。安装器也接受 `0.4.0` 或 `v0.4.0` 这样的数字参数。

2026-09-26 之前安装的程序可能无法识别新的章节标签。请重新运行上面的安装命令，
以取得支持章节标签的更新器；即使程序版本号相同，也需要替换一次。
安装与升级保留已有配置、SOUL 和历史，下载或校验失败时保留旧程序。

| 章节 | 程序版本 | Git 标签 |
| --- | --- | --- |
| 第 01 章 | `0.1.0` | [`chapter-01`](https://github.com/qshine/mino/releases/tag/chapter-01) |

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
- [全部章节规划](docs/books/zh/plan-todo-chapters.md)
- 启动入口：`cmd/mino/main.go`；应用实现和测试：`internal/`。
- `internal/app.go` 负责组装，`internal/gateway/` 负责终端交互，
  `internal/agent/` 负责模型请求；配置和 SOUL 保留在 `internal/`。
- SDK 负责 API 通信，Mino 负责终端交互；工具执行和 Agent 循环仍属于后续章节。
- [贡献约定](AGENTS.md) · [MIT 许可证](LICENSE)

## 图文教程书

[在线阅读中文版](https://qshine.github.io/mino/zh/) · [Read online in English](https://qshine.github.io/mino/)

书籍源码分别位于 `docs/books/en/` 和 `docs/books/zh/`，网站默认英文，可切换简体中文。
章节聚焦 Agent 交互，安装配置和发布细节放在配套页面中。
使用 Node.js 24，在仓库中运行 `npm ci --ignore-scripts`、`npm run book:dev`，
即可打开输出的本地地址预览。`npm run book:build` 检查站内链接并构建两种语言。

项目专用 `book_writer` subagent 会在 Codex 完成代码修改后维护教程。
GitHub Actions 会在 `main` 上的相关内容更新后检查书籍并发布到 GitHub Pages。
上方在线链接直接打开教程网站。
详见[写作与发布流程](docs/books/zh/maintaining-the-book.md)。
