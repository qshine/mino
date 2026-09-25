# 第 03 章：工具调用与 Bash Agent 循环

状态：**已实现并发布**（2026-09-26）。目标版本：`chapter-03`。前置：[JSONL 历史](./02-jsonl-history.md)。公共约定见[计划索引](./README.md)。

## 1. 本章交付

用户问 `Which Go version is installed here?`。模型提出 Bash 调用，Mino 展示待执行命令并请求批准，执行后把结果交还模型，模型据此回答。完成一次“请求 → 调用 → 结果 → 再请求”的 Agent 循环。

本章只有一个 Bash 工具，不做并行执行、后台任务、自动重试或通用插件框架。Bash 运行在用户账户权限下，逐次确认不等于操作系统沙箱。

## 2. 工具定义与循环状态

`bash` 的参数只包含必需的字符串 `command`，使用对象 JSON Schema，拒绝未知字段。工作目录、环境变量、超时和输出限额由程序选择，不让模型在参数中自行扩大。

Gateway 在 `internal/gateway/` 中接收问题、显示事件和收集确认。Agent 在 `internal/agent/agent.go` 中通过 `Handle` 保存用户消息，再进入直接调用 SDK 的 `runLoop`；`Session` 管理内存状态，`history.go` 管理 JSONL 文件。工具在 `internal/tools/` 实现统一 `Tool` 接口（`Definition`、`Prepare`、`Execute`），由启动入口通过构造函数注入。每次完整模型响应及工具结果写入并同步成功后，才更新内存并继续。

```text
Persist user → Request model → Collect and validate complete response
  ├─ final text, no calls → Persist response + turn_end → Prompt
  └─ calls → Persist response → Validate and ask → Persist start
              → Execute → Persist result → Request model again
```

模型输出可以同时包含文本和调用。文本继续流式显示，但有调用时还没有结束用户回合。只有收到完整响应后才验证并执行调用，绝不执行流中尚未拼完的参数。没有文本但有有效调用是合法中间响应；既无文本也无调用应报错。

一个响应有多个调用时按返回顺序逐个处理，每个调用分别批准。循环最多 8 次模型请求、每回合最多 16 个工具调用；达到限额后停止，补齐尚未执行调用的结果状态，显示 `Tool limit reached.`，不通过额外模型请求绕过上限。

## 3. 协议与持久化

保留第 02 章 `v: 1` 的读取方式，新记录写为 `v: 2`。旧版程序无法读取含新记录的文件，回退前须恢复私有备份。以下是本地日志字段，不是 Responses API 的直接参数表：

| 记录 | 必要内容 | 作用 |
| --- | --- | --- |
| `model_response` | `turn_id`、完整且可回放的输出项 `output` | 保留文本、函数调用和服务要求回传的其他输出项及原顺序 |
| `tool_start` | `turn_id`、`call_id`、工具名、已校验参数、实际 cwd | 记录即将开始一次已批准的执行；同步成功后才能启动进程 |
| `tool_result` | `turn_id`、`call_id`、嵌套 `result`（状态、输出、退出码、截断标记） | 与具体调用配对；失败和拒绝也是结果 |
| `turn_end` | 成功、失败、取消或中断状态 | 标识用户回合如何停止，不替代工具结果 |
| `recovery_ack` | `turn_id` | 持久化对未知结果的知情确认；不授予未来执行许可 |

`call_id` 从模型调用原样保存，与本地回合 ID 分开；在整个用户回合内检查唯一性。同一响应出现重复或缺失 ID，或后续响应复用已处理的 ID 时，拒绝整个新批次，不执行其中一部分，也不重新执行旧调用。这类协议无效响应不写成可回放的 `model_response`，只记录净化后的失败状态，避免留下无法配对的历史。未知工具、参数非法和用户拒绝则可用有界的错误结果回传给模型，不能当作 Bash 命令尝试执行。

每次请求以 SDK 支持的类型把先前的模型输出及对应 `function_call_output` 放回 `input`。已用当前 SDK 和本地模拟服务验证输出转输入、加密推理输出项和 `store: false` 的无状态回放，显式申请 `reasoning.encrypted_content`。真实模型与兼容端点仍需用户在其配置下验证。不能丢掉必要项，也不能记录或展示服务未提供的内部推理。兼容端点不支持该协议时清楚报错，不偷偷降级为文本拼接。

完整模型响应同步后才能运行工具；每次工具结果同步后才能再请求模型。重启只恢复记录，不再次执行历史调用。

## 4. 授权与 Bash 执行

1. 以明确的工具面板显示工具名、转义后的完整命令、真实 cwd、超时和输出上限。不能仅用 `terminalText` 删除控制字符后让显示命令与执行命令看起来一致。
2. 请求一次性确认，如 `Approve this operation? [y/N]`；默认拒绝。批准只适用于当前已解析的参数，参数、目录或环境发生变化必须重新确认。
3. 终端读取由一个位置负责。聊天与批准不能各自创建 stdin scanner；非交互输入默认不能批准工具，测试通过注入的确认函数回答。
4. 执行 `/bin/bash --noprofile --norc -c <command>`，固定本次启动的 cwd；使用明确的最小环境，移除 `BASH_ENV` 等启动注入变量，不把 API 配置传给子进程。固定环境仅含 `PATH`、`HOME` 和 `LANG=en_US.UTF-8`。PATH 为 `/opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin`；其他安装位置的程序须使用绝对路径。
5. 默认 30 秒超时，stdout 与 stderr 合计最多 64 KiB。超过上限终止进程组，回传保留的有界输出与 `truncated: true`；同时处理进程结束、管道关闭和读取协程退出。
6. 在 macOS 为进程设置独立进程组。取消、超时或超量时终止该组并等待清理；自行脱离进程组的子进程仍是限制，需要如实说明，不能称为完整隔离。
7. 输出在模型上下文中标为工具数据；显示时过滤控制字符。非零退出码也是一个可解释的工具结果，不必让整个聊天程序崩溃。

不提供“批准所有命令”。命令可以读写用户有权限的文件，也可能联网；没有 OS 沙箱时，Mino 不能仅靠关键词或 cwd 限制这些能力。第 09 章再讨论受限 Bash 或结构化文件工具。

## 5. 中断与恢复

工具有外部副作用，不能继续沿用“失败回合一律不进入上下文”的简单规则。把已闭合、能合法回放的失败回合也纳入上下文，并明确标注中断；部分模型文本不伪装成完成回答。

| 恢复时的记录 | 处理 |
| --- | --- |
| 调用存在，无 `tool_start` 和结果 | 补充 `not_executed` 结果；不自动执行 |
| 有开始记录，无结果 | 结果标为 `unknown`；操作可能已经发生。进入恢复提示，用户确认了解不确定性后才继续聊天，禁止自动重跑 |
| 有完整结果，无最终回答 | 回放已有结果，记录回合中断；用户下一次输入才能触发新请求 |
| 多调用批次只完成部分 | 为其余调用分别生成 `not_executed` 或 `unknown`，保持每个 ID 都有对应结果 |
| 执行后保存结果失败 | 停止聊天并说明结果未可靠保存；不得为了拿到结果再执行一次 |
| 用户取消或批准时 EOF | 未启动的调用记为取消/未执行；已启动的终止并收集结果；不再请求模型 |

`unknown` 是事实上的未知结果，不伪造退出码或成功文本。恢复状态不发放任何未来许可。服务无法接受本地恢复结果的表现形式时，应在协议适配测试中发现，并在继续前明确停止。

## 6. 任务顺序与验收

| 任务 | 依赖 | 实际文件 | 验收与验证 |
| --- | --- | --- | --- |
| 03-A：验证完整调用回放 | 02 | `internal/agent/agent.go`、`internal/agent/tool_protocol_test.go`、`internal/agent/streaming_test.go` | 模拟文本、调用及其他必要输出项；只在完整响应后产出调用；下一请求保留顺序和 ID |
| 03-B：接入单工具循环与日志 | A | `internal/agent/agent.go`、`internal/agent/agent_test.go`、`internal/agent/history.go`、`internal/agent/session.go`、`internal/agent/history_tools_test.go` | 使用假执行器走完调用后回答；多调用、未知工具、拒绝和循环上限均有配对结果 |
| 03-C：实现受限额的 Bash 执行器 | B | `internal/tools/tool.go`、`internal/tools/bash.go`、`internal/tools/bash_test.go` | 临时目录测试 stdout/stderr、非零退出、超时、输出洪泛和子进程清理；不读真实用户文件 |
| 03-D：接入授权交互 | C | `internal/gateway/cli.go`、`internal/gateway/approval_test.go`、`internal/app.go` | 默认拒绝时假执行器调用数为零；修改参数需新批准；聊天与批准没有竞争读 stdin |
| 03-E：故障恢复 | D | `internal/agent/agent_integration_test.go`、`internal/agent/session.go`、`internal/agent/history_tools_test.go` | 在开始前、执行后、结果保存后模拟退出；重启执行计数不增加，未知结果必须提示 |

检查点：A–B 先用假工具演示；C–D 在临时目录演示批准与拒绝；E 后核对重启回放。完成时运行 `go test ./...`、`bash scripts/check.sh`，再更新双语教程并执行 `npm run book:build`。

## 7. 教程与后续接口

书中只围绕一次查询 Go 版本的交互解释 `tools`、调用 ID、执行和结果回传，再用拒绝执行证明控制权在程序和用户。工具进度显示与最终回答应能区分。原始日志可在下一章整体分配给一个会话。

## 8. 实现决定与验证边界

- 用户要求将 Bash 与未来工具统一放在 `internal/tools/` 目录；使用轻量 Tool 接口和名称索引，不引入消息总线或依赖注入框架。
- 每次调用单独批准；默认采用 30 秒、64 KiB、8 次模型请求和 16 次工具调用。第 8 次响应仍请求工具、或新批次令总调用数超过 16 时，不执行该批次，补齐 `not_executed` 并停止。
- 固定启动目录与上述最小环境；授权显示与实际执行使用同一实例。模型不能通过参数改变运行目录、环境或限额。
- 未知结果使用启动确认提示，拒绝或 EOF 便退出。确认写入 `recovery_ack`，重启不会绕过未确认状态，也不会重跑历史命令。
- 已通过 `go test ./...` 和 `bash scripts/check.sh`（包括 vet、竞态测试和构建）。真实 Bash 与模拟模型的集成测试只在临时目录写入标记文件，并验证重启回放不执行旧命令。未使用真实 API 密钥或付费模型；模拟协议验证不保证任意兼容端点均支持。
- 默认 SOUL 已更新为工具能力；已有用户 SOUL 保留。旧文件中的“不能执行工具”描述需由用户参考新默认值手动更新。

## 9. Gateway / Agent 重构验收

- Gateway 只依赖 Agent 的 `Handler` / `Interaction` 契约，不读写会话文件；Agent 不导入 Gateway。
- `Handle` 先持久化用户消息，`runLoop` 集中展示 SDK 请求、流式处理、完整响应保存及工具执行。每次完整响应保存，片段只显示。
- 当前仍是一份 `history.jsonl`；内存 Session 与日志同步，不新增多会话命令。
- 测试注入第二个工具，验证定义进入请求、执行前用户消息/模型响应/开始记录已保存、下一请求前结果已保存；无需改 Agent 循环。
- 同一 Session 拒绝并发回合；写入或 Sync 失败不提交内存；既有 v1/v2 回放、恢复、批准及限额测试继续通过。
- 实现文件收敛为 Gateway 两个、Agent 三个、Tools 两个；入口显式组装依赖。
