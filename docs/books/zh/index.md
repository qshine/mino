---
next:
  text: 准备与安装
  link: /zh/getting-started
---

# 用 Go，从零构建一个 Agent

**从一次交互理解 Agent 的工作方式，用代码和实验验证原理。**

Mino 是一个用 Go 从零构建 Agent（智能体）的项目，这本教程记录它逐章增加能力的过程。你会从终端问答开始，看清模型收到什么、程序负责什么，再亲手检验每一步的结果和边界。

**当前可读：**[第 01 章：与模型对话](./chapters/01-terminal-chat.md) · [第 02 章：JSONL 对话历史](./chapters/02-jsonl-history.md) · [第 03 章：Agent 循环与 Bash](./chapters/03-tools-and-bash.md) · [第 04 章：恢复中断的工具调用](./chapters/04-recovering-tool-turns.md) · [第 05 章：多会话](./chapters/05-jsonl-sessions.md) · [第 06 章：上下文压缩](./chapters/06-context-compaction.md) · [第 07 章：Skills](./chapters/07-skills.md) · [第 08 章：文件工具](./chapters/08-file-tools.md) · [第 09 章：MCP 工具](./chapters/09-mcp.md) · [第 10 章：安全防护](./chapters/10-guardrails.md) · [第 11 章：评测](./chapters/11-evals.md) · [第 12 章：沙箱](./chapters/12-sandbox.md)。第 01–12 章均已发布，最新一章对应 `chapter-12`（版本 `0.12.0`）。第 13–15 章处于规划中，见[章节规划](./plan-todo-chapters.md)。

## 关于作者

<div class="author-card">

**qqling | AI Builder**：资深开发者，做过大厂和 Startup；AI 资深玩家，专注大模型与 Agent 落地。我想从零构建一个属于自己的 Agent，Mino 就是这个想法的起点。每增加一种能力，书里就说明它解决什么问题、怎样工作，以及如何验证结果，让刚接触 Agent 的读者也能一步步跟上。

- GitHub：[qshine](https://github.com/qshine)
- 微信公众号：qqling的AI小院
- 小红书：[AI_qqling](https://www.xiaohongshu.com/user/profile/67039740000000001d031479)

<AuthorQr wechat="微信公众号：qqling的AI小院" wechat-alt="微信公众号“qqling的AI小院”二维码" xiaohongshu="小红书：AI_qqling" xiaohongshu-alt="小红书“AI_qqling”二维码" />

</div>

## 适合谁读

这本书面向刚接触 Agent、希望动手理解其原理的读者。阅读源码需要了解 Go 的变量、函数和基本错误处理；第一次接触模型 API 也可以从第 01 章开始。HTTP 请求、上下文、工具调用等概念，会在真正需要它们时展开。

跟练环境是 macOS。下载运行 Mino 无需安装 Go，从源码构建或运行 Mino 时需要安装 Go；具体步骤见[准备与安装](./getting-started.md)。

## 每章会给你什么

1. **观察一次交互。** 从输入问题、看到回答等具体动作开始，明确这一章要解决的问题。
2. **看清模型与程序的分工。** 配合图解和对应版本的源码，追踪输入、上下文与响应如何流动。
3. **用实验验证。** 检查请求内容或运行模拟测试，确认能力来自哪里、在什么情况下会失效。

例如，第 01 章通过检查请求说明为什么同一个终端里的两次提问没有记忆，第 02 章再验证保存的历史怎样进入下一次请求。模型碰巧答对，不能替代对程序行为的检查。

## 当前可以读到哪里

第 01–10 章于 **2026-10-21** 以 `chapter-01` 至 `chapter-10` 发布，程序版本为 `0.1.0` 至 `0.10.0`；`chapter-10` 于 **2026-10-26** 重新发布，第 11、12 章随后于 **2026-10-26** 以 `chapter-11`（版本 `0.11.0`）和 `chapter-12`（版本 `0.12.0`）发布。第 10 章把此前的控制统一成一个安全防护决定，并提供由你选择的权限模式；第 11 章用声明好的评测用例衡量 Agent；第 12 章在 macOS 上让每条获批的 Bash 命令都在操作系统沙箱内运行。各已发布章节注明了实验所需的源码标签。

| 阶段 | 你要理解的问题 |
| --- | --- |
| [01 与模型对话](./chapters/01-terminal-chat.md) | 一行问题如何变成 HTTP 请求，再变成终端里的回答？ |
| [02 JSONL 对话历史](./chapters/02-jsonl-history.md) | 哪些内容能在重启后保留，哪些记录会成为模型上下文？ |
| [03 Agent 循环与 Bash](./chapters/03-tools-and-bash.md) | 命令由谁批准和执行？结果怎样回到模型？ |
| [04 恢复中断的工具调用](./chapters/04-recovering-tool-turns.md) | 命令可能已经执行、结果却没保存下来时，模型怎样得知？ |
| [05 多会话](./chapters/05-jsonl-sessions.md) | 怎样选择一段对话的上下文、保留另一段，并确认清空当前会话？ |
| [06 上下文压缩](./chapters/06-context-compaction.md) | 手动与自动压缩如何共用预算，并保留有用的上下文？ |
| [07 Skills](./chapters/07-skills.md) | 固定目录怎样引导模型用 `skill_view` 请求任务知识？ |
| [08 文件工具](./chapters/08-file-tools.md) | 程序怎样限定模型读取项目文件的范围，并把每次获批的写入绑定到你看到的文件？ |
| [09 MCP 工具](./chapters/09-mcp.md) | 远端服务或本地进程提供的工具怎样加入 Agent 循环？每次批准决定的是什么？ |
| [10 安全防护](./chapters/10-guardrails.md) | 怎样把实际权限统一成一套由代码执行的策略？ |
| [11 评测](./chapters/11-evals.md) | 怎样可重复地检查 SOUL、工具描述或循环的一处改动是否改变了 Agent 的行为？ |
| [12 沙箱](./chapters/12-sandbox.md) | 怎样让内核而不是你对命令的解读，去限定一条获批的 Bash 命令能触及什么？ |

## 如何配合代码阅读

已发布章节会注明适用版本，并使用对应的 `chapter-NN` 源码标签。这一版书中最新的第 12 章对应 [chapter-12](https://github.com/qshine/mino/tree/chapter-12)，[源码准备说明](./getting-started.md#从源码体验第十二章)介绍了怎样在本地运行同一份快照。

发布标签使用 `chapter-NN`，程序版本采用 `0.章节.修订号`。例如，补丁标签 `chapter-05.1` 生成版本 `0.5.1`，并更新对应章节的说明，无需新增一章。

使用过早期 `0.1.0` 或 `0.2.0` 构建的读者，请按[更新说明](./getting-started.md#查看版本和更新)操作。

从[准备与安装](./getting-started.md)开始，或直接阅读[第 01 章](./chapters/01-terminal-chat.md)。
