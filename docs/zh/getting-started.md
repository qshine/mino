---
prev:
  text: 关于这本书
  link: /zh/
next:
  text: 01 与模型对话
  link: /zh/chapters/01-terminal-chat
---

# 准备与安装

先把 Mino 跑起来。第一章再沿着代码解释启动、请求和回答的完整路径。

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

设置保存在 `~/.mino/config.json`。之后配置齐全就直接进入聊天；升级不会覆盖这些设置。密钥以明文保存在本地，目录和文件分别使用 `0700`、`0600` 权限。

在 `You>` 后输入问题即可。真实问答会请求你配置的服务，并按该服务的规则计费。只输入 `/exit` 可以验证程序能够启动，无需调用模型。

## 查看版本和更新

```bash
mino version
mino update
```

更新仍需 GitHub 账号的仓库访问权限。下载、附件校验失败时，保留现有程序；已有模型配置继续使用。

当前目录的 `AGENTS.md` 可以为模型提供项目指令。这个文件是可选的，安装后的 Mino 可以在没有项目文件的目录中启动。

## 从源码学习

若要运行或修改源码，需要 Go 1.27.1 或更高的兼容工具链。克隆仓库后，从仓库目录运行：

```bash
gh repo clone qshine/mino
cd mino
go run .
```

安装包和源码运行共用用户目录中的配置。自动化测试使用临时目录和模拟模型服务，不需要真实 API Key：

```bash
bash scripts/check.sh
```

接下来进入[第一章：与模型对话](./chapters/01-terminal-chat.md)，看清一次问答里程序与模型各自负责什么。
