# 第 01 章：终端对话基线

状态：**已实现基线，供后续章节对照**。适用章节版本：`0.1.x`；本次核对代码：`chapter-01`。参见[现有教程](../books/en/chapters/01-terminal-chat.md)与[计划索引](./README.md)。本文件不安排重写本章。

## 1. 目标与边界

用户在 macOS 终端输入一个问题，Mino 立即显示模型返回的文本增量，确认响应完成后再接受下一问。每一问独立，只有当前输入和本次启动读取的身份说明进入请求；没有历史、会话、模型工具执行或压缩。

## 2. 已有实现

| 位置 | 当前职责 | 后续演进时要保留的行为 |
| --- | --- | --- |
| [`internal/gateway/cli.go`](https://github.com/qshine/mino/blob/chapter-01/internal/gateway/cli.go) | 聊天入口、版本、帮助和更新命令 | 版本和帮助不要求完成 API 配置 |
| [`internal/app.go`](https://github.com/qshine/mino/blob/chapter-01/internal/app.go) | 共用输入缓冲区，完成配置，加载身份，连接终端与客户端 | 首次配置不能吞掉后续聊天输入 |
| [`internal/config.go`](https://github.com/qshine/mino/blob/chapter-01/internal/config.go) | 读取和补齐 API URL、模型、密钥 | 从用户 HOME 定位；目录 `0700`，文件 `0600`；配置取消不改旧设置 |
| [`internal/soul.go`](https://github.com/qshine/mino/blob/chapter-01/internal/soul.go) | 首次创建并读取用户 SOUL | 启动读取一次，保留用户编辑；拒绝符号链接、空白、无效 UTF-8 和超 64 KiB 文件 |
| [`internal/agent/responses.go`](https://github.com/qshine/mino/blob/chapter-01/internal/agent/responses.go) | 官方 SDK 发起流式 Responses 请求 | `store: false`；不跟随重定向，不自动重试，响应上限 8 MiB，请求超时 2 分钟 |
| [`internal/gateway/terminal.go`](https://github.com/qshine/mino/blob/chapter-01/internal/gateway/terminal.go) | 输入循环、文本显示、退出和取消 | 空行不请求；`/exit`、EOF、Ctrl+C 退出；过滤显示中的终端控制字符 |

现有客户端的调用形状为 `respond(ctx, prompt, emit) error`：`prompt` 是单个字符串，`emit` 负责显示增量，没有返回可持久化的回答对象。终端对每个输入调用一次它；这个循环还不是工具 Agent 循环。

## 3. 一次交互的状态变化

1. CLI 无参数时进入 `run`；配置完整则直接加载身份，否则仅询问缺失字段。
2. 终端读取并整理输入，处理退出和空行，打印 `Assistant>`。
3. 客户端提交 `model`、`instructions`、当前 `input` 和 `store: false`。
4. `response.output_text.delta` 立即显示；拒绝响应的增量也显示，并只添加一次拒绝提示。
5. 只有完整的 `response.completed`、成功状态且有可见内容，才判定本次成功。流提前结束、失败或不完整均报错。
6. 普通请求失败后可继续下一问；已经显示的部分文本留在终端，但不表示回答完整。全局取消或输出失败结束运行。

交互示意，模型措辞不固定：

```text
You> Remember the word pine.
Assistant> ...
You> What word did I give you?
Assistant> ...
```

记忆边界由模拟服务捕获第二次请求来验证：其中没有第一问及其回答。不能仅凭模型是否猜中 `pine` 判断有无历史。

## 4. 已有验证与后续回归

已有 `TestRespondSendsIndependentRequests`、`TestRunStreamsBeforeResponseCompletes`、`TestRespondHonorsCancellationAndTimeout`、`TestTerminalKeepsPartialAnswerAndContinuesAfterStreamFailure` 等测试，分别验证独立请求、增量时序、取消与部分输出。配置和 SOUL 的测试隔离用户目录。

开下一章前运行：

```bash
go test ./...
bash scripts/check.sh
```

第 02 章有意改变“请求独立”的预期，应将相应测试改为验证历史内容和顺序。其余流式显示、服务错误不泄露密钥、输入取消、私有文件和环境变量隔离等约束继续保留。

## 5. 交给下一章的改动点

- 为 `respond` 增加显式上下文输入和完成结果返回值；终端仍通过回调显示增量。不要把终端打印出来的字符串当作回答事实来源。
- 在 `app.go` 组装历史存储和对话控制逻辑，把文件读写移出 HTTP 客户端和终端渲染函数。
- 引入一条 JSONL 历史后，更新启动提示及默认 SOUL 的能力说明，同时保留用户已经编辑的 SOUL。

## 6. 下一次讨论需要确认

第 01 章无需重新决策。下一章主要讨论“哪些未完成输入保留在日志、哪些进入下次请求”和“尾部损坏如何恢复”，具体建议在[第 02 章草案](./02-jsonl-history.md)。当前 Ctrl+C 退出整个程序；是否增加“只取消当前生成”属于另一个交互选择，不能在加入历史时顺带改变。
