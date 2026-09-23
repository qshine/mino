# Mino

[English](README.md) | 简体中文

用 Go 从零实现终端 Agent 的分章教程。第一章通过 OpenAI Responses API 实现独立的一问一答。
支持 **macOS 13 及以上版本**，提供 **Apple Silicon 和 Intel Mac** 安装包。

## 安装

使用安装包不需要安装 Go。仓库处于私有状态时，需要有 `qshine/mino` 访问权限的 GitHub 账号，
并安装 [GitHub CLI](https://cli.github.com/)。如果已安装 Homebrew，可先运行 `brew install gh`，再登录一次：

```bash
gh auth login --hostname github.com
```

然后在 Bash 或 zsh 中执行这一行命令：

```bash
mino_installer="$(gh api --hostname github.com -H 'Accept: application/vnd.github.raw+json' 'repos/qshine/mino/contents/install.sh?ref=main')" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
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

配置包含 `base_url`、`api_key`、`model`。密钥以明文保存在本地，目录权限为 `0700`，
文件权限为 `0600`。修改配置可编辑该文件；将某字段清空，重启时会重新询问。
程序不读取项目内的配置、`.env` 或 `OPENAI_*` 环境变量。
如果用过旧版项目内的 `miniagent.json`，可在新配置尚不存在时将它迁移到新位置。

自定义服务必须支持 `/responses`。API 地址填写前缀，通常以 `/v1` 结尾，远程服务要求 HTTPS。
当前目录有 `AGENTS.md` 时，会将其作为模型指令；没有该文件也能正常启动。

输入问题后按回车。使用 `/exit`、空行上的 Ctrl+D 或 Ctrl+C 退出。第一章不保留聊天历史。

## 版本与升级

```bash
mino version          # 查看当前版本
mino update           # 安装最新发布版本
mino update v0.1.0    # 安装指定版本，也可用于回退
```

升级会更新 `~/.mino/bin/mino`，复用 GitHub 登录状态，保留已有配置。
下载或校验失败时不会替换旧程序。

| 进度 | 版本号 | Git 标签 |
| --- | --- | --- |
| 第一章 | `0.1.0` | `v0.1.0` |
| 第一章小修正 | `0.1.1`、`0.1.2` | `v0.1.1`、`v0.1.2` |
| 第二章 | `0.2.0` | `v0.2.0` |
| 第二章小修正 | `0.2.1` | `v0.2.1` |

推送 `main` 会触发 CI。推送版本标签会触发测试、两种 Mac 架构的编译和 GitHub Release 发布，
同时上传校验文件。发布版的版本号来自 Git 标签，直接从源码运行时显示 `dev`。
已经发布的标签和安装包保持不变，修复时发布新的补丁版本。
详见[发布说明](docs/releases.md)和[更新记录](CHANGELOG.md)。

## 开发与学习

开发环境使用 **Go 1.27.1** 和标准库，没有第三方 Go 依赖。
已安装 Go 1.21+ 且使用默认 `GOTOOLCHAIN=auto` 时，可以自动下载所需工具链。

在仓库根目录执行：

```bash
go run .
bash scripts/check.sh
```

检查脚本会验证格式和 Shell 语法，运行 `go vet`、完整测试和竞态检测，并生成 `bin/mino`。
测试使用假密钥、临时用户目录、本机模拟 HTTP 服务和模拟下载，不调用付费模型，也不修改真实配置。

- [第一章：问题、实现、演示与验证](docs/chapters/01-terminal-chat.md)
- [全部章节规划](docs/plan-todo-chapters.md)
- 阅读顺序：`main.go` → `cli.go` → `config.go` / `config_prompt.go` → `terminal.go` → `responses.go`。
- [贡献约定](AGENTS.md) · [MIT 许可证](LICENSE)
