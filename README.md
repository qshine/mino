# Miniagent

用 Go 从零实现终端 Agent 的分章教程。当前已实现[第 01 章：与模型对话](docs/chapters/01-terminal-chat.md)，通过 OpenAI Responses API 完成独立的一问一答，并读取项目根目录的 `AGENTS.md` 作为模型指令。

## 在 macOS 终端启动

要求 macOS 13 或更新版本，支持 Apple Silicon 和 Intel Mac。项目使用 **Go 1.27.1**（2026-09-20 核实的[最新稳定版](https://go.dev/dl/)），没有第三方依赖。

在项目根目录打开终端，直接启动：

```zsh
go run .
```

第一次启动时，程序会询问缺失的配置：

```text
API URL [https://api.openai.com/v1]:
Model:
API Key (input hidden):
```

程序提示全部使用英文。地址可直接按回车使用 OpenAI 官方默认值，也可以填写自己的服务地址。模型名称没有默认值，必须填写；留空会重新询问。API Key 同样必填，输入时不会回显。启动时自动创建用户主目录下的 `~/.mino/`，配置齐全后保存到 `~/.mino/config.json`，立即进入对话；以后直接 `go run .` 就会读取配置并进入对话。仅缺部分字段时，只询问缺失项。

`~/.mino/config.json` 中保存 `base_url`、`api_key`、`model` 三个字段。密钥以明文保存在本地；目录权限为 `0700`，文件权限为 `0600`，仅当前用户可访问。配置位置不随工作目录变化，升级或替换项目代码也不会覆盖它。修改配置可直接编辑该文件；将某字段设为空字符串，重启时会再次询问。程序以此文件为准，不读取工作目录中的配置、`OPENAI_*` 环境变量或 `.env` 文件。取消首次设置时会保留目录，但不写入不完整的配置。

如果此前使用项目内的 `miniagent.json`，可以在新配置尚不存在时将原文件迁移到 `~/.mino/config.json`；否则首次启动会重新询问。不要用旧文件覆盖已配置好的新文件。

自定义地址必须支持 `/responses`，`base_url` 填 API 前缀（通常以 `/v1` 结尾），程序会追加 `/responses`。模型名称请填写服务实际支持且账户有权限使用的名称。

若尚未安装 Go，从[官方安装页](https://go.dev/doc/install)安装对应架构的 macOS 包。已安装 Go 1.21+ 且保持默认 `GOTOOLCHAIN=auto` 时，首次运行会根据 `go.mod` 自动下载所需工具链，需要网络；这不会替换系统原有的 Go 安装。

输入问题后按回车；空行跳过；输入 `/exit`、在空输入行按 Ctrl+D，或按 Ctrl+C 退出。每次问题独立，不保留聊天历史。也可以编译后启动：

```zsh
go build -o miniagent .
./miniagent
```

两种方式均须从项目根目录运行，以读取这里的 `AGENTS.md`。

## 开发与学习

```zsh
go fmt ./...
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

测试使用本机模拟 HTTP 服务，不需要真实密钥，不调用付费模型。工具链下载完成后，测试不需要外部网络。

- [第一章：问题、实现、演示与验证](docs/chapters/01-terminal-chat.md)
- [全部章节规划及完成状态](docs/plan-todo-chapters.md)
- 代码阅读顺序：`main.go` → `config.go` / `config_prompt.go` → `terminal.go` → `responses.go`；所有文件属于同一个 `main` 包。
- 贡献约定见 [AGENTS.md](AGENTS.md)。请勿提交真实密钥、会话数据库或私密日志。
