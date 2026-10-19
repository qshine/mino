# 第 10 章：用评测衡量 Agent

状态：**已实现并发布**（`chapter-10`，版本 `0.10.0`，2026-10-19）。前置：第 03 章的工具循环、第 07 章的文件工具、第 09 章的策略。公共约定见[计划索引](./README.md)。

## 0. 已确认的范围（2026-10-16）

- **交付形态**：完整的第 10 章。新增 `internal/eval` 包、`mino eval` 子命令、测试、仓库级 `evals/` 用例、中英文教程第 10 章，以及 README、CHANGELOG、路线图和侧边栏更新。
- **两种模式**：默认 **replay**，用一个脚本化的本机 Responses 服务驱动真实的 `agent.Handle`，确定性、不需要密钥、不访问网络，纳入 `scripts/check.sh`；显式 `--live` 使用 `~/.mino/config.json` 里的真实模型，每个用例跑 N 次（默认 3），报告通过率、pass@k、pass^k 和 token 用量。
- **只做确定性 grader**：`final_text_contains`、`final_text_matches`、`tool_called`、`tool_not_called`、`tool_order`、`result_status`、`file_exists`、`file_contains`、`max_requests`、`max_tool_calls`。不做 LLM judge，教程里说明原因。
- **不做**：OS 沙箱、MCP 连接、读取用户 `~/.mino/policy.json`（评测固定使用默认策略以保证可复现）、按参数匹配的断言、费用定价、成对对照臂。

## 1. 调研：其他 Agent 框架怎么做

调研日期 2026-10-16，只记录能在公开仓库或文档里核实的事实。

### 1.1 pi（badlogic/pi-mono）

- **确定性循环测试**：`packages/ai` 提供 faux provider（`fauxProvider()`、`fauxToolCall(name, args)`、`setResponses`），把模型响应脚本化；`packages/agent/test/agent-loop.test.ts` 用它断言事件序列、执行了哪些工具 id、`isError`、串并行，以及"被截断的 assistant 消息里的工具调用不执行"。`packages/coding-agent/test/suite/` 的会话测试必须走 faux provider，不得使用真实密钥。来源：<https://github.com/badlogic/pi-mono>（`packages/ai/README.md`、`packages/agent/test/agent-loop.test.ts`、`packages/coding-agent/test/suite/README.md`）。
- **评测套件**：`packages/evals` 基于 vitest-evals。用例是 TypeScript，一文件一个 `describeEval`，每个 case 调 `run(prompt)`，返回 `{output, events[], usage{tokens, cost}, timings}`；每次运行使用独立 HOME、工作区和 agent 目录，并把 session JSONL 存成产物；`--runs-per-variant N` 重复运行；"docs lift" 评测在两个 Docker 容器里成对比较有无文档的差异，缺一臂的配对不计入通过率。来源：`packages/evals/README.md`、`src/harness.ts`。
- **live 矩阵**：`packages/ai/test/cross-provider-handoff.test.ts` 列出数十个 provider/model，`describe.skipIf(!hasAnyApiKey())` 门控；`test.sh` 用 `env -i` 加白名单运行，密钥不会泄漏给普通测试。
- **用量**：每条 assistant 消息带 `usage{input, output, cost}`，定价来自 models.dev 生成的目录。作者在博客中说这"够个人使用，不够精确计费"。来源：<https://mariozechner.at/posts/2025-11-30-pi-coding-agent/>。

### 1.2 Hermes Agent（NousResearch/hermes-agent）

- **没有统一的 `hermes eval` 命令**：issue #44000 提议 `hermes eval list/run/history` 与 code/model/human grader，截至调研仍为 open。来源：<https://github.com/NousResearch/hermes-agent/issues/44000>。
- **`evals/` 是一组临时 A/B 脚本**：如 `evals/toolperf_abeval/` 用 9 个陷阱任务对比基线与修复两臂，指标包括 LLM 轮次、工具调用数、工具错误、重试、字节数和耗时，成功判定靠 marker 字符串和磁盘状态；README 明示 "n=3 下的成功率差异是噪声"。`evals/token_accounting/replay_gates.py` 让真实 `AIAgent` 循环跑在本机假 OpenAI 服务上，按场景输出 PASS/FAIL。
- **测试里的假模型**：`tests/fakes/fake_llm_provider.py` 是"脚本化、可录制的 loopback LLM provider"，讲 OpenAI Chat Completions（JSON 与 SSE），能脚本化文本、工具调用、错误和挂起，并记录每个请求体。
- **轨迹与打分分离**：`batch_runner.py` 批量产出 trajectory JSONL（ShareGPT 格式、`tool_stats`、`api_calls`），没有分数字段；打分在 Atropos 环境的 `compute_reward` 里（例如测试通过 1.0、文件存在 0.5、否则 0.0）。来源：<https://hermes-agent.nousresearch.com/docs/user-guide/features/batch-processing>、`environments/README.md` 镜像。

### 1.3 其他参考

| 来源 | 借鉴点 |
| --- | --- |
| Claude Code `claude plugin eval` | 用例目录 = `prompt.md` + `graders/*.md`，确定性 grader 有 `regex`、`tool_used`、`tool_order`、`file_exists`，LLM judge 另列；默认每例 3 次；与无插件基线对比 |
| Inspect AI（UK AISI） | Task = Dataset + Solver + Scorer；`--epochs N` 配 `pass_at(k)`、`pass_k(k)` reducer；`.eval` 日志含模型用量 |
| tau-bench | 成功 = 最终数据库状态与目标一致，与轨迹无关；pass^k 的无偏估计为 C(c,k)/C(n,k) |
| Terminal-Bench / Harbor | 任务目录 = 指令 + `tests/test.sh` 写出数字 reward；`-k` 次尝试 |
| Anthropic《Demystifying evals》 | 术语 task/trial/grader/transcript；代码 grader 优先；"评结果，不评路径"；pass@k 与 pass^k 的区别 |

### 1.4 对 Mino 的结论

1. 用**脚本化假服务 + 真实 Agent 循环**取得确定性（pi faux provider、Hermes FakeLLMServer 的共同做法），而不是 mock Agent 内部。
2. 用例是**文件**，先做确定性 grader（正则、工具是否被调用、顺序、结果状态、文件状态），LLM judge 延后。
3. **每次运行独立目录**，把 session JSONL 当作轨迹产物保存（pi 的做法），Mino 的 v4 记录天然就是轨迹。
4. **N 次运行 + pass@k / pass^k**，k 取运行次数；用量从 `response.completed.usage` 读取，不做定价。
5. live 模式**显式开启、密钥只来自 `~/.mino/config.json`**，CI 永不触发。

## 2. 用例文件

`mino eval [PATH]`，PATH 是单个文件或目录（非递归，`*.yaml`/`*.yml` 按文件名排序），默认 `./evals`。用例是项目数据，和 `.agents/skills` 一样由用户主动运行；它不是运行配置，所以不违反"不读项目配置"的约定。绝不从 `~/.mino` 加载用例。

```yaml
name: bash-denied            # 必填，须等于文件名去掉扩展名；规则同技能名
description: Without approval the command never runs.
prompt: Delete the build directory.
instructions: ...            # 可选，替代默认指令
approve: none                # all | none | [bash, write_file]，默认 all
runs: 2                      # 可选，覆盖 --runs（1..32）
workspace:
  files:
    notes.md: "hello\n"      # 相对路径，≤64 个文件，每个 ≤64 KiB
model:                       # replay 脚本，≤16 步；--live 时可省略
  - call: {tool: bash, arguments: {command: "rm -rf build"}}
  - text: The command was not approved, so nothing was deleted.
    expect_input_contains: did not approve   # 对请求体 JSON 的子串断言
graders:                     # 1..32 条，全部通过才算这一次运行通过
  - tool_called: bash
  - result_status: {call: 1, status: denied}
  - final_text_contains: not approved
  - max_requests: 2
```

解析使用 yaml.v3 并开启 `KnownFields`，未知字段、两个键的 grader、不能编译的正则、越界路径都在加载时报错。用例文件 ≤64 KiB，一个目录 ≤256 个用例。

## 3. 包与命令

- `internal/eval/case.go`：`Case`、`Approval`、`Step`、`Grader` 与 `Load`/`Parse`。
- `internal/eval/fake.go`：`FakeModel`，监听 `127.0.0.1:0`，只接受 `POST /v1/responses`，按脚本逐步返回 `response.completed`（`message` 或 `function_call` 项，`call_id` 全局唯一），附带按"4 字节一个 token"合成的 `usage`；脚本耗尽返回 `response.failed` 并记录原因；`expect_input_contains` 失败也记录。
- `internal/eval/run.go`：`RunCase` 每次运行创建 `mino-eval-*` 临时根目录，下设兄弟目录 `workspace/`、`home/.mino/`、`session/`；工具为 Bash（cwd=workspace，HOME=home）、`read_file`/`write_file`/`edit_file`（限定 workspace）、`skill_view`（扫描 workspace 的 `.agents/skills`）；`agent.OpenSession` + `agent.New`（默认策略），用实现了 `agent.Interaction` 的记录器按 `approve` 回答确认。结束后读取该运行的 `history.jsonl` 重建调用列表（`model_response` 里的每个 `function_call`、是否有 `tool_start`、配对的 `tool_result`），再评分，再删除目录。`--out` 时把日志复制为 `<out>/<case>/run-N.jsonl`。
- `internal/eval/grade.go`：`Grade(Case, Transcript) []Outcome`，文件类 grader 用 `os.OpenRoot(workspace)` 读取。
- `internal/eval/report.go`：`Summarize`、`PassAtK = 1 − C(n−c,k)/C(n,k)`、`PassPowK = C(c,k)/C(n,k)`、文本表格与 `report.json`。
- `internal/eval_command.go`：`mino eval [--live] [--runs N] [--filter NAME] [--out DIR] [--timeout D] [PATH]`。replay 默认指令为内嵌 `SOUL.md`，完全不触碰 `~/.mino`；`--live` 用 `loadConfig(nil)`（不提示，配置不完整即报错）和 `~/.mino/SOUL.md`。任一用例失败返回错误，退出码 1。
- Agent 的最小改动：`agent.Event` 新增 `response_completed` 类型与 `Usage` 字段，`runLoop` 在校验完成的响应后发出；终端网关忽略它。历史记录仍是 v4，不保存用量。

## 4. 报告与指标

```text
Eval replay mode: 5 cases, 1 runs each, scripted model (synthetic token counts).
CASE             RUNS  PASS  pass@k  pass^k  REQ  TOOLS  IN_TOK  OUT_TOK  STATUS
bash-approved    1     1     1.00    1.00    2.0  1.0    3828    71       pass
protected-env    1     1     1.00    1.00    3.0  2.0    5880    117      pass
All cases passed.
```

k 等于运行次数：pass@k 回答"是否至少有一次通过"，pass^k 回答"是否每次都通过"。`--runs 1` 时两列退化为通过/失败。token 在 replay 模式是合成值，只用来验证统计路径；live 模式才是服务返回的真实计数。

## 5. 验证记录（2026-10-17）

- `go test -race ./internal/eval ./internal/ ./internal/agent ./internal/gateway` 通过；`internal/eval` 覆盖 schema 校验、假服务的项形状与违约、批准/拒绝/顺序/文件/技能/超时用例、pass@k/pass^k 数学与报告输出；`internal/eval_command_test.go` 覆盖混合套件退出码、`--filter`、`--out`、各类非法参数、`--live` 缺配置与 httptest 服务下的 live 运行。
- 构建二进制后在临时 HOME 下运行 `bin/mino eval ./internal/eval/testdata`（一通过一失败，退出码 1）、`bin/mino eval ./evals`（五例全部通过）；运行后临时 HOME 下不存在 `.mino`。
- `scripts/check.sh` 末尾新增 `bin/mino eval ./evals`。本容器为 Linux，`internal/tools` 里既有的 Bash 进程组清理测试在改动前后同样失败，与本章无关；项目 CI 在 macOS 上运行。

## 6. 待定问题

- 是否需要成对对照（例如有无某条 SOUL 指令）的 lift 报告；当前通过两个用例文件手工比较。
- 是否把 live 模式的 token 汇总换算成费用；需要维护模型定价表，暂不做。
- 是否录制 live 运行为 replay 脚本（从 `model_response` 记录生成 `model:`）；实现简单，但要先决定录制后的用例如何保持可读。
