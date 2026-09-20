# 第 01 章：与模型对话——第一个终端程序

## 这一章解决什么问题

模型不会直接读取你的键盘，也不会自动把回答显示在终端。我们先搭通一条完整路径：终端读取一行问题，Go 构造 HTTP 请求，模型生成回答，Go 解析回答并打印，然后等待下一次输入。

本章交付一个能在 macOS 终端运行的程序，使用 Go 1.27.1 和标准库。首次启动交互填写缺失的 `base_url`、`api_key`、`model`，保存到用户主目录 `~/.mino/config.json`，之后自动复用。启动时读取根目录 `AGENTS.md`，每轮传入 Responses API 的 `instructions`。每个请求仅包含当前问题，各轮之间没有上下文关联，也不创建本地聊天记录。内存历史会在第 02 章加入。

## 1. 准备 macOS 和 Go

本章面向 macOS 13+ 的 Apple Silicon（`arm64`）和 Intel（`amd64`）Mac，使用系统自带的终端和默认 zsh 即可。

截至 2026-09-20，[Go 官方下载页](https://go.dev/dl/)列出的最新稳定版是 **1.27.1**。在官方页面选择对应架构的 `.pkg` 并按[安装说明](https://go.dev/doc/install)安装；安装后重新打开终端。

如果已经有 Go 1.21+，默认的 `GOTOOLCHAIN=auto` 可以按项目要求自动下载工具链。进入本项目根目录，执行：

```zsh
go version
```

本章验证使用的输出为 `go version go1.27.1 darwin/arm64`；Intel Mac 的架构为 `darwin/amd64`。`go.mod` 的 `go 1.27.1` 设置最低工具链版本，更高版本也可使用。自动下载机制见 [Go Toolchains](https://go.dev/doc/toolchain)。如果此前关闭了自动选择，可以为当前命令设置 `GOTOOLCHAIN=auto go run .`，或手动安装所需版本。

## 2. 配置模型服务

在项目根目录直接启动，不需要先设置环境变量：

```zsh
go run .
```

如果配置文件尚不存在，会看到以下提示：

```text
Complete the missing settings. Press Enter to accept a value in brackets, or Ctrl+C to cancel. Settings will be saved to ~/.mino/config.json.
API URL [https://api.openai.com/v1]:
Model: your-model
API Key (input hidden):
Settings saved. Next time, chat will start immediately.
```

| 配置字段 | 默认值与填写方式 |
| --- | --- |
| `base_url` | 回车接受 `https://api.openai.com/v1`，或输入自己的 API 前缀 |
| `model` | 无默认值，必须输入服务支持且账户有权限的模型名称；空白会重新询问 |
| `api_key` | 无默认值，必须输入；输入不回显，空白会重新询问 |

程序的配置、聊天和错误提示全部使用英文。上面的 `your-model` 是占位示例，请填写实际模型名称。只有 API 地址提供默认值，模型名称和密钥都必须由用户填写。

启动时自动创建 `~/.mino/` 目录；填写完成后，程序会先保存 `config.json` 再进入问答。下一次运行 `go run .` 或 `./miniagent` 时，配置完整就直接进入对话；如果只缺少 `api_key`，就只询问密钥。配置文件结构如下，示例中的密钥是占位符：

```json
{
  "base_url": "https://api.openai.com/v1",
  "api_key": "你的密钥",
  "model": "your-model"
}
```

配置统一读取 `~/.mino/config.json`，不随当前工作目录变化。不读取项目内的 `miniagent.json`、`config.json`、旧版的 `OPENAI_*` 环境变量或 `.env`。旧版项目配置需要在新配置不存在时手动迁移到新位置，或者在首次启动时重新填写。要修改服务或模型，可以编辑 JSON 后重启；把字段设为 `""` 可让程序重新询问这一项。JSON 损坏会明确报错并保留原文件。输入过程中按 Ctrl+C 或在空行按 Ctrl+D 会取消设置，保留原有配置，不保存尚未填写完整的内容；首次创建的 `.mino` 目录会保留。

密钥以明文保存在这个本地文件中。程序将 `.mino` 目录权限设为 `0700`，配置文件权限设为 `0600`，只允许当前用户访问。配置文件和写入时的临时文件都位于用户目录中，项目升级不会覆盖它们；`.gitignore` 仍忽略旧版项目配置，避免误提交旧密钥。配置不作为聊天内容传给模型，密钥只用于 HTTP 认证。完整配置表示字段齐全且格式有效，实际密钥和模型权限由服务端校验。

如使用兼容服务，把 `base_url` 改成该服务的 API 前缀，例如 `https://gateway.example.com/v1`，不要填 `/chat/completions` 或完整的 `/responses` 地址。远程连接要求 HTTPS；本机模拟服务可以使用 `http://127.0.0.1:端口/v1`。程序不跟随重定向，避免把问题或密钥转发到别处。

## 3. 从输入走到输出

整个过程可以沿着以下源文件阅读：

```text
main.go：读取 AGENTS.md，加载或补全配置，监听 Ctrl+C
  ├─ config.go / config_prompt.go：读取 JSON → 只询问缺失项 → 保存 JSON
  └─ terminal.go：读入一行非空问题
       └─ responses.go：POST {base_url}/responses
            └─ 提取回答 → 终端打印 → 再读一行
```

### 入口与配置：`main.go`、`config.go`、`config_prompt.go`

`configPath` 通过 `os.UserHomeDir()` 定位用户主目录，创建或保护 `.mino` 目录，不允许目录是普通文件或符号链接。`loadConfig` 从该目录读取 JSON，保留已有字段，再依次补全缺失的地址、模型和密钥。只有地址提示中的方括号显示默认值，空输入会采用该地址；模型和密钥没有默认值，必须填写。URL 和必填项校验通过后，在同一个 `.mino` 目录中使用权限为 `0600` 的临时文件写入，再重命名为 `config.json`，避免写入中断留下半份配置。读取时拒绝目录和符号链接，已有文件的权限也会收紧到 `0600`。

`config_prompt.go` 用 macOS 自带的 `/bin/stty` 暂时关闭密钥输入回显，并用 `defer` 恢复原终端状态，覆盖成功、EOF 和 Ctrl+C 退出路径。Go 仍只依赖标准库；这里调用的是固定的终端设置命令，参数不包含用户输入。配置和聊天共用同一个输入缓冲区，避免配置阶段预读的第一条问题丢失。

`loadInstructions` 从**当前工作目录**读取 `AGENTS.md`，因此必须从仓库根目录启动。文件内容在启动时读取一次，作为每一轮的 `instructions`；修改文件后需要重启生效。文件缺失或无法读取会导致启动失败，而不会悄悄丢弃项目指令。

这里选择标准库 `net/http` 和 `encoding/json`，让读者能看见完整的请求构造与响应解析。所有文件仍在同一个 `main` 包中，只按职责拆分文件，暂时不引入 SDK 或其他包层次。

### 构造请求：`responses.go`

请求发送到 `{base_url}/responses`，头部包含 `Authorization: Bearer ...` 和 `Content-Type: application/json`。JSON 主体形如：

```json
{
  "model": "你配置的模型名称",
  "instructions": "AGENTS.md 的完整内容",
  "input": "用一句话解释什么是 Agent。",
  "store": false
}
```

`instructions` 是 Responses API 提供的高优先级指令字段；`input` 字符串表示当前用户问题。请求中没有历史消息、`previous_response_id` 或 `conversation`，因此下一轮无法从程序获得上一轮内容。`store: false` 请求服务不要保存可供后续查询的响应对象；它不等于对服务端所有日志或数据保留策略的承诺。协议参见 [Responses 创建接口](https://developers.openai.com/api/reference/python/resources/responses/methods/create)。

程序只将 `AGENTS.md` 与当前问题发送给你配置的服务。`AGENTS.md` 是你主动提供的项目指令，请不要在其中放密钥。本章没有 Bash、文件修改或其他工具执行能力；模型回答仅被显示为文本。

### 解析回答

Responses API 的 `output` 是一个数组，里面可能先出现推理条目，文本不一定在第一项。程序遍历 `type == "message"` 且 `role == "assistant"` 的条目，再连接其中所有 `output_text` 文本。遇到 `refusal` 时显示拒绝内容。

不能照搬某些 SDK 的顶层 `output_text` 便捷属性去解析原始 HTTP JSON。这里按照[官方文本生成说明](https://developers.openai.com/api/docs/guides/text)读取 `output[].content[]`。无效 JSON、空文本、失败状态以及未完成的响应都会给出错误，不会假装已有完整回答。

### 终端交互：`terminal.go`

`bufio.Scanner` 按行读取输入，每行是一条独立问题，空白行跳过。程序等待完整响应后一次性打印，本章不做流式输出。

扫描输入使用一个 goroutine，主循环通过 `select` 同时等待输入和取消信号。`signal.NotifyContext` 将 Ctrl+C 转成取消通知；HTTP 请求也携带同一个 context，因此无论正在等键盘还是等网络，都能退出。阻塞在标准输入上的扫描由进程退出结束；复用这段函数处理其他输入流时，调用者负责关闭输入流。

| 操作 | 结果 |
| --- | --- |
| 输入文字并按回车 | 发送一个独立请求 |
| 空白行 | 不调用 API，继续等待 |
| `/exit` | 正常退出 |
| Ctrl+D（空输入行） | 收到 EOF 后退出 |
| Ctrl+C | 取消当前请求并退出程序 |
| 请求失败 | 向标准错误输出提示，继续等待下一次输入 |

每个请求最多等待两分钟。输入单行须小于 1 MiB，HTTP 响应上限为 8 MiB。API 错误正文不直接输出，避免服务回显密钥；模型回答中的终端控制字符会被过滤，保留换行和制表符。程序不会自动重试付费请求。

## 4. 观察运行结果

下面是交互形式的示意，真实措辞由模型决定：

```text
Miniagent - Chapter 01: Terminal Chat
Each question is independent. Use /exit, Ctrl+D, or Ctrl+C to quit.

You> 本项目只支持哪一种模型 API？

Assistant> 本项目只支持 OpenAI Responses API。

You> 我最喜欢的颜色是蓝色。

Assistant> 好的。

You> 我刚才说最喜欢什么颜色？

Assistant> 当前问题没有提供你喜欢的颜色。

You> /exit
Goodbye.
```

第一个问题用于观察 `AGENTS.md` 的作用。后两个问题用于理解单轮边界：最后一轮的 HTTP 请求只有“我刚才说最喜欢什么颜色？”。模型可能猜测答案，因此应以请求内容和自动化测试为准，不能只凭它是否答对判断是否携带历史。

编译后可以直接运行二进制，仍须保留根目录为当前目录：

```zsh
go build -o miniagent .
./miniagent
```

配置完成后，也支持按行的管道输入。配置不完整时必须先在交互终端补全，程序会报错，避免把管道里的聊天问题误存为密钥：

```zsh
printf '用一句话解释 Agent。\n/exit\n' | go run .
```

## 5. 验证与排错

从项目根目录运行：

```zsh
go fmt ./...
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

自动化测试通过隔离的临时用户目录验证默认值、配置保存、切换工作目录后复用配置、忽略项目配置、再次启动不提问、只补缺失字段、目录与文件权限、符号链接拒绝、无效主目录、损坏文件保护、取消设置及输入缓冲。测试不会读写开发者真实的 `~/.mino`。通过 `httptest` 模拟 Responses 服务，覆盖项目指令注入、两轮独立请求、全部文本片段提取、拒绝回答、HTTP 错误、超时/取消、输入与响应限制、终端退出，以及不生成本地历史文件。测试密钥是固定的假值，不访问真实模型；需要允许进程绑定本机临时端口。

本章在 Go 1.27.1、macOS 26.3.2 / Apple Silicon 上通过构建、完整测试、竞态检测和 `go vet`。伪终端连接本机模拟服务验证了中文问答、HTTP 429 后继续输入、`/exit`、Ctrl+D、等待输入或响应时的 Ctrl+C，以及 `go run .` 管道输入。Intel Mac 版本通过交叉编译，未在 Intel 实机运行；本次验收未调用真实模型服务。

2026-09-23 的配置交互验收另外验证了英文提示、默认 API 地址、模型没有默认值且留空重试、错误地址和空密钥重试、密钥不回显、用户目录中的配置保存与权限、切换工作目录后直接进入聊天、只补缺失字段，以及 Ctrl+C/Ctrl+D 取消后恢复终端且不保存配置。所有密钥均为测试假值。

| 现象 | 检查方法 |
| --- | --- |
| 提示配置不完整且需要交互终端 | 先直接运行 `go run .` 补全配置，再使用管道输入 |
| `~/.mino/config.json` 格式错误 | 检查 JSON 引号、逗号和字段类型；原文件不会被覆盖 |
| 修改模型或密钥 | 编辑 `~/.mino/config.json`，或把相应字段设为空字符串后重启 |
| 读取 `AGENTS.md` 失败 | 切回仓库根目录，再运行 `go run .` 或 `./miniagent` |
| HTTP 401/403 | 检查密钥、模型权限以及密钥是否属于当前服务 |
| HTTP 404 | 检查 API 前缀与模型名，确认服务支持 Responses API |
| HTTP 429 | 检查额度或限流，稍后手动重试 |
| HTTP 3xx | 填入服务最终 API 地址；程序不会跟随重定向 |
| 响应未完成或没有文本 | 缩短问题或检查兼容服务实际返回的协议 |
| 网络错误、超时 | 检查服务地址、网络及本机代理配置 |
| Go 工具链下载失败 | 检查网络，或从官方页面安装 Go 1.27.1+ |

本章完成后，终端到模型的路径已经可运行。第 02 章将在此基础上维护内存历史，使模型能收到先前的问题和回答。
