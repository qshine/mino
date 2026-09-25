---
prev:
  text: 关于这本书
  link: /zh/
next:
  text: 01 与模型对话
  link: /zh/chapters/01-terminal-chat
---

# 准备与安装

本页准备于 2026-09-26 发布的 `chapter-01`（程序版本 `0.1.0`）。本章包含独立的终端问答。运行各章实验时，请使用该章注明的标签。

## 准备一台 Mac

支持 macOS 13 及以上，提供 Apple Silicon 和 Intel 两种安装包。安装器会自动选择对应的版本，不需要你先安装 Go。

仓库和发布包现已公开。安装使用 macOS 自带的 `curl`，不需要 GitHub CLI，也不需要登录 GitHub。首次启动配置模型服务时，才需要填写模型 API Key。

## 一行安装

在 Bash 或 zsh 中运行：

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-01 && export PATH="$HOME/.mino/bin:$PATH"
```

此命令从 `main` 获取当前安装器，并选择 `chapter-01`。省略 `-- chapter-01` 即可安装最新发布版。安装器从同一个已确定的发布标签下载 macOS 安装包和校验文件。

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

设置保存在 `~/.mino/config.json`。之后配置齐全就直接进入聊天；升级不会覆盖这些设置。密钥以明文保存在本地，目录和文件分别使用 `0700`、`0600` 权限。

修改服务、模型或密钥时，编辑这个文件后重启。空字段会重新询问；设置过程中按 Ctrl+C，已有配置保持不变。Mino 不读取项目内配置、`.env` 或 `OPENAI_*` 环境变量。不要把配置文件放进仓库。

在 `You>` 后输入问题即可。真实问答会请求你配置的服务，并按该服务的规则计费。只输入 `/exit` 可以验证程序能够启动，无需调用模型。

## 查看版本和更新

如果 Mino 安装于章节标签迁移之前，请先把上面的安装命令**重新运行一次**，即使 `mino version` 已显示 `0.1.0`。旧的内嵌更新器无法解析 `chapter-*` 标签。重新安装会保留配置、自定义 `~/.mino/SOUL.md` 和已有历史。

安装章节标签对应的发布版后，可以检查数字版本，再次选择本章：

```bash
mino version
mino update chapter-01
```

不带参数的 `mino update` 选择最新发布版。新版安装器也接受 `0.1.0` 或 `v0.1.0`，两者都指向 `chapter-01`。下载或校验失败时保留现有程序。详见[发布与补丁规则](./releases.md#版本规则)。

## Mino 的身份

配置齐全后，Mino 在进入聊天前读取一次 `~/.mino/SOUL.md`。文件缺失时，程序用内嵌的默认身份创建它。已有内容会在重启和更新时保留；配置未完成或被取消时不会创建这个文件。

默认内容把 Mino 描述为终端助手，可以帮助回答问题、解释概念、写作，以及讨论用户提供的代码，并要求模型使用你的语言、如实说明限制。修改用户文件可以调整这些指引，重启 Mino 后生效；程序不支持即时重载。完整文本会通过 `instructions` 发送给配置的模型服务，不要在其中放置密钥。

文件必须是普通文件，包含非空的 UTF-8 文本，大小不超过 64 KiB。目录、符号链接或无效文本会在发送模型请求前被拒绝。目录和文件权限分别为 `0700`、`0600`。Mino 忽略工作目录中的 `AGENTS.md` 和 `SOUL.md`：仓库的 `AGENTS.md` 用于开发，根目录的 `SOUL.md` 仅提供构建时嵌入的默认内容。

## 从源码学习

若要运行或修改源码，需要 Go 1.27.1 或更高的兼容工具链。克隆仓库后，从仓库目录运行：

```bash
git clone https://github.com/qshine/mino.git
cd mino
git checkout chapter-01
go run ./cmd/mino
```

`cmd/mino/main.go` 是可执行程序的启动入口。终端交互位于 `internal/gateway/`，模型请求位于 `internal/agent/`；应用组装、配置和身份加载保留在 `internal/`。测试与对应代码相邻，内嵌安装器和默认身份不依赖源码目录。

模块文件仍在仓库根目录。`go.mod` 将官方 `github.com/openai/openai-go/v3` SDK 固定为 v3.66.0，`go.sum` 记录依赖校验值。首次构建时，Go 会下载依赖，无需单独安装 SDK。

安装包和源码运行共用用户目录中的配置。自动化测试使用临时目录和模拟模型服务，不需要真实 API Key：

```bash
bash scripts/check.sh
```

接下来进入[第一章：与模型对话](./chapters/01-terminal-chat.md)，看清一次问答里程序与模型各自负责什么。
