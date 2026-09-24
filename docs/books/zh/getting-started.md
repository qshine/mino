---
prev:
  text: 关于这本书
  link: /zh/
next:
  text: 01 与模型对话
  link: /zh/chapters/01-terminal-chat
---

# 准备与安装

这一页负责准备好 Mino。第一章再追踪一条问题从终端输入到模型回答的完整路径。

## 准备一台 Mac

支持 macOS 13 及以上，提供 Apple Silicon 和 Intel 两种安装包。安装器会自动选择对应的版本，不需要你先安装 Go。

仓库目前是私有的。下载需要一个有仓库访问权限的 GitHub 账号，以及 [GitHub CLI](https://cli.github.com/)。如果使用 Homebrew，可以先运行 `brew install gh`，然后登录一次：

```bash
gh auth login --hostname github.com
```

GitHub 登录用于下载程序，下一步填写的模型 API Key 用于访问模型服务；它们是两套独立的凭据。

## 一行安装

在 Bash 或 zsh 中运行：

```bash
mino_installer="$(gh api --hostname github.com -H 'Accept: application/vnd.github.raw+json' 'repos/qshine/mino/contents/install.sh?ref=main')" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
```

安装器校验下载包的 SHA-256 和程序版本，将可执行文件放入 `~/.mino/bin/mino`，并设置终端的查找路径。首次安装会创建 `~/.mino`，配置文件会在首次启动填写完成后创建。

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

自定义服务必须支持 Responses API。填写 API 前缀，例如 `https://gateway.example.com/v1`，不要加上 `/responses` 或 `/chat/completions`。远程地址必须使用 HTTPS。

设置保存在 `~/.mino/config.json`。之后配置齐全就直接进入聊天；升级不会覆盖这些设置。密钥以明文保存在本地，目录和文件分别使用 `0700`、`0600` 权限。

修改服务、模型或密钥时，编辑这个文件后重启。空字段会重新询问；设置过程中按 Ctrl+C，已有配置保持不变。Mino 不读取项目内配置、`.env` 或 `OPENAI_*` 环境变量。不要把配置文件放进仓库。

在 `You>` 后输入问题即可。真实问答会请求你配置的服务，并按该服务的规则计费。只输入 `/exit` 可以验证程序能够启动，无需调用模型。

## 查看版本和更新

```bash
mino version
mino update
```

更新仍需 GitHub 账号的仓库访问权限。下载、附件校验失败时，保留现有程序；已有模型配置继续使用。

重新发布的 `v0.1.0` 包含官方 SDK、根目录安装脚本和用户 `SOUL.md` 身份。如果安装过早期第一章构建，即使 `mino version` 已显示 `0.1.0`，也要运行 `mino update v0.1.0` 重装，因为版本号没有变化。如果旧版更新器失败，请使用上面的安装命令。两种方式都会保留配置和自定义 `~/.mino/SOUL.md`。详见[第一章基线重置说明](./releases.md#第一章基线重置)。

## Mino 的身份

配置齐全后，Mino 在进入聊天前读取一次 `~/.mino/SOUL.md`。文件缺失时，程序用内嵌的默认身份创建它。已有内容会在重启和更新时保留；配置未完成或被取消时不会创建这个文件。

默认内容把 Mino 描述为终端助手，可以帮助回答问题、解释概念、写作，以及讨论用户提供的代码，并要求模型使用你的语言、如实说明限制。修改用户文件可以调整这些指引，重启 Mino 后生效；程序不支持即时重载。完整文本会通过 `instructions` 发送给配置的模型服务，不要在其中放置密钥。

文件必须是普通文件，包含非空的 UTF-8 文本，大小不超过 64 KiB。目录、符号链接或无效文本会在发送模型请求前被拒绝。目录和文件权限分别为 `0700`、`0600`。Mino 忽略工作目录中的 `AGENTS.md` 和 `SOUL.md`：仓库的 `AGENTS.md` 用于开发，根目录的 `SOUL.md` 仅提供构建时嵌入的默认内容。

## 从源码学习

若要运行或修改源码，需要 Go 1.27.1 或更高的兼容工具链。克隆仓库后，从仓库目录运行：

```bash
gh repo clone qshine/mino
cd mino
go run ./cmd/mino
```

`cmd/mino/main.go` 是可执行程序的启动入口。应用代码及对应测试放在 `internal/` 下的 `mino` 包中，由 `app.go` 将启动流程接到终端循环。阅读时从 `app.go` 开始，再沿 `terminal.go` 进入 `responses.go`。

入口以 `mino` 为别名导入 `github.com/qshine/mino/internal`，因此调用仍是 `mino.Main(version)`。

根目录的 `assets.go` 嵌入 `install.sh` 和默认 `SOUL.md`。`internal/cli.go` 使用 `assets.InstallerScript` 执行更新，`internal/soul.go` 使用 `assets.DefaultSoul` 初始化用户身份；两者都不依赖源码目录。

模块文件仍在仓库根目录。`go.mod` 将官方 `github.com/openai/openai-go/v3` SDK 固定为 v3.66.0，`go.sum` 记录依赖校验值。首次构建时，Go 会下载依赖，无需单独安装 SDK。

安装包和源码运行共用用户目录中的配置。自动化测试使用临时目录和模拟模型服务，不需要真实 API Key：

```bash
bash scripts/check.sh
```

接下来进入[第一章：与模型对话](./chapters/01-terminal-chat.md)，看清一次问答里程序与模型各自负责什么。
