# 发布 Mino 应用

[英文介绍](https://github.com/qshine/mino#readme) · [中文介绍](https://github.com/qshine/mino/blob/main/README.zh-CN.md)

这里说明 Go 应用的版本发布。书籍网站的构建和发布见[写作与更新流程](./maintaining-the-book.md)。

## 版本规则

教程采用 `0.章节.修订号`：第一章是 `0.1.0`，第一次修复是 `0.1.1`，第二章从 `0.2.0` 开始。Git 标签带有 `v` 前缀。

发布工作流通过 Go 链接参数把标签版本写入可执行文件。源码开发构建显示 `dev`，无需手动修改源码中的版本常量。

## 发布一个版本

1. 完成本章或修复，在 `CHANGELOG.md` 添加条目并提交。
2. 运行 `bash scripts/check.sh`。审核改动，然后推送到 `main`。
3. 创建并推送下一个版本标签，例如：

   ```bash
   git tag -a v0.1.2 -m 'Mino v0.1.2'
   git push origin v0.1.2
   ```

[Release 工作流](https://github.com/qshine/mino/blob/main/.github/workflows/release.yml) 会重新检查代码，并在关闭 CGO 的情况下构建 `darwin/arm64` 和 `darwin/amd64`。打包时会运行本机 Apple Silicon 程序验证版本；Intel 程序通过交叉编译生成。

发布附件包含可执行文件和 MIT 许可证：

```text
mino_0.1.2_darwin_arm64.tar.gz
mino_0.1.2_darwin_amd64.tar.gz
checksums.txt
```

工作流先创建草稿 Release，全部附件上传成功后才正式发布。用户随后通过 `mino update` 获得新版本。发布说明来自对应的 changelog 条目；没有匹配条目的标签会在打包时失败。普通分支推送运行 [CI](https://github.com/qshine/mino/blob/main/.github/workflows/ci.yml)，不会发布应用版本。

GitHub 提供构建机器和下载存储，不需要自备服务器。私有仓库的构建会使用账号的 GitHub Actions 配额。发布任务使用内置的 `GITHUB_TOKEN` 和 `contents: write` 权限，无需把个人访问令牌或模型 API Key 加入 Actions secrets。

## 本地打包与恢复

发布前可以先检查安装包：

```bash
bash scripts/package.sh v0.1.1
```

选择已经记录在 `CHANGELOG.md` 中的版本。结果写入被 Git 忽略的 `dist/` 目录；本地打包不会创建标签或 Release。

工作流失败时先检查日志。附件上传失败可能留下草稿；重新运行发布任务前，应检查并删除对应的未完成草稿。不要移动已经发布的标签，也不要替换已发布附件；修复应获得新的补丁版本号。

用户可以通过 `mino update v0.1.1` 重装某个已发布版本。这只替换可执行文件，不会回滚或删除 `~/.mino/config.json`。

## 私有仓库安装

README 的安装命令通过已经登录的 `gh api` 从 `main` 读取 `install.sh`。安装器和内嵌的更新器使用 `gh` 获取 Release ID，再通过 GitHub 专门的附件接口下载文件，避免版本元数据中不完整的附件列表阻止安装。

用户必须登录有权读取仓库的账号。令牌仍由 GitHub CLI 管理，不会复制到 Mino 配置中。

安装会创建 `~/.mino/bin/mino`。首次启动询问模型配置，全部填写完成后才写入 `~/.mino/config.json`。安装器向 `.zshrc`（支持 `ZDOTDIR`）或 `.bash_profile` 添加 PATH 行，不覆盖原文件。如果配置文件是符号链接或不可写，会打印手动添加的方法。README 的安装命令还会更新当前终端的 PATH。
