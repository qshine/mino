# 发布 Mino 应用

本页说明应用发布。书籍网站见[写作与更新流程](./maintaining-the-book.md)。

第 01–05 章于 **2026-09-26** 以 `chapter-01` 至 `chapter-05` 发布，第 06–08 章于 **2026-10-11** 以 `chapter-06` 至 `chapter-08` 发布，第九章于 **2026-10-12** 以 `chapter-09` 发布，第十章于 **2026-10-19** 以 `chapter-10` 发布，均提供 Apple Silicon 和 Intel 的 macOS 安装包。最新章节发布版为 [chapter-10](https://github.com/qshine/mino/releases/tag/chapter-10)，程序版本 **0.10.0**；各章使用对应的源码标签。

第五章写入版本 3 的会话记录，读取版本 1–3。体验或回退前，请查看[历史兼容性说明](./getting-started.md#第五章的历史兼容性)。构建当前工作区不会创建发布版，也不会改变已发布标签。

第六章写入版本 4 记录，读取版本 1–4；使用已有会话或回退前，请查看[第六章兼容性说明](./getting-started.md#第六章的历史兼容性)。

第七章沿用记录版本 4，但 `chapter-06` 无法加载启动过文件工具调用的会话；回退前请查看[第七章兼容性说明](./getting-started.md#第七章的历史兼容性)。

第八章同样沿用记录版本 4，但 `chapter-07` 无法加载开始过 MCP 调用的会话；回退前请查看[第八章兼容性说明](./getting-started.md#第八章的历史兼容性)。

第九章沿用记录版本 4，`chapter-08` 仍能打开它的会话；见[第九章兼容性说明](./getting-started.md#第九章的历史兼容性)。

第十章沿用记录版本 4，没有新增记录类型；`chapter-09` 仍能打开它的会话，但不包含 `mino eval`。见[第十章兼容性说明](./getting-started.md#第十章的历史兼容性)。

## 版本规则

发布标签标识章节，可执行文件和压缩包名称保留数字版本：

| 发布标签 | 程序版本 |
| --- | --- |
| `chapter-01` | `0.1.0` |
| `chapter-02` | `0.2.0` |
| `chapter-03` | `0.3.0` |
| `chapter-04` | `0.4.0` |
| `chapter-05` | `0.5.0` |
| `chapter-06` | `0.6.0` |
| `chapter-07` | `0.7.0` |
| `chapter-08` | `0.8.0` |
| `chapter-09` | `0.9.0` |
| `chapter-10` | `0.10.0` |

章节初版使用 `chapter-NN`，后续修复使用 `chapter-NN.PATCH`。例如，`chapter-04.1` 生成版本 `0.4.1`。后续修复创建新标签，不移动已发布标签或替换附件。每章尽量保留一次完整提交，包含代码、测试和双语书稿。

打包时从标签换算出 `0.章节.修订号`，通过 Go 链接参数写入程序。源码构建显示 `dev`。Changelog 标题继续使用数字版本，例如 `## [0.4.1]`。

## 迁移已有安装

**章节标签迁移前安装的更新器无法解析 `chapter-*` 标签。** 请先把[当前安装命令](./getting-started.md#一行安装)重新运行一次，即使程序显示的数字版本没有变化。这会获取新版安装器并替换程序，保留配置、自定义 `~/.mino/SOUL.md` 和已有历史。修改标签或 `main` 上的脚本不会改变已安装的程序。

完成这次安装后，`mino update` 选择最新发布版，`mino update chapter-10` 选择本章。安装器也接受 `0.10.0` 或 `v0.10.0`，两者都指向 `chapter-10`；这些是版本选择别名，不是额外的 Git 标签。

## 发布一个版本

1. 完成改动，在 `CHANGELOG.md` 添加数字版本条目，并提交。
2. 运行 `bash scripts/check.sh`，审核代码和双语书稿，再把已审核的改动推送到 `main`。
3. 创建并推送新的章节或补丁标签。以下为第 10 章的未来补丁示例：

   ```bash
   git tag -a chapter-10.1 -m 'Mino chapter-10.1'
   git push origin chapter-10.1
   ```

[Release 工作流](https://github.com/qshine/mino/blob/main/.github/workflows/release.yml) 由 `chapter-*` 标签触发，重新运行检查，并在关闭 CGO 的情况下构建 `darwin/arm64` 和 `darwin/amd64`。在 macOS 上打包时，还会运行当前主机架构对应的程序，验证数字版本。

每个压缩包包含 `mino`、`LICENSE` 和 `THIRD_PARTY_NOTICES.txt`。上面的补丁示例会提供：

```text
mino_0.10.1_darwin_arm64.tar.gz
mino_0.10.1_darwin_amd64.tar.gz
checksums.txt
```

工作流先上传到草稿 Release，全部附件上传成功后才发布。对应数字版本的 changelog 条目提供发布说明；缺少条目会导致打包失败。普通分支推送运行 CI，不发布应用。发布任务使用 GitHub 内置的 `GITHUB_TOKEN` 和 `contents: write` 权限，不需要模型 API Key。

## 本地打包与恢复

在 `chapter-10` 的仓库根目录检查本章安装包：

```bash
bash scripts/package.sh chapter-10
```

打包接受规范的章节标签，使用当前工作区源码，不会检出标签。结果写入被 Git 忽略的 `dist/`，不创建 Git 标签或 Release。发布失败时，检查日志及可能残留的未完成草稿，再重新运行任务。

## 公开下载与内嵌更新器

安装器使用 `curl`，无需 GitHub CLI 或 GitHub 登录。除非指定版本，否则只解析一次最新标签，然后从同一标签下载压缩包与 `checksums.txt`。校验 SHA-256 和程序数字版本后，才替换 `~/.mino/bin/mino`；下载或校验失败时保留现有程序。

各章发布包都内嵌了 `mino update` 使用的安装器。安装会通过 `.zshrc`（支持 `ZDOTDIR`）或 `.bash_profile` 配置 PATH，不覆盖配置文件。遇到符号链接或不可写的配置文件时，会打印需要手动添加的行。首次启动完成设置后，才创建模型配置文件。
