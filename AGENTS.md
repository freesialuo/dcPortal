# AGENTS.md

本文件用于规范在本仓库内工作的“人类协作者 + 自动化代理（AI Agents）”行为，确保改动可追踪、可复现、可发布。

## 目标

- 保持 DCPortal 在“安全可控安装”场景下的稳定性。
- 优先保证 OAuth2 流程与安装治理功能正确性。
- 所有改动都应可测试、可回滚、可审查。

## 仓库关键上下文

- 主程序入口：`cmd/dcportal/main.go`
- 业务处理：`internal/handler`
- Discord API：`internal/discord`
- 数据层：`internal/store`
- 配置层：`internal/config`
- 前端模板：`web/templates`
- 静态样式：`web/static`

## 协作原则

- 先理解需求，再改代码；避免“猜测式”大改。
- 尽量最小改动闭环解决问题。
- 不要在无必要情况下重构无关模块。
- 不要提交密钥、token、真实凭据。

## 代码改动规范

- Go 代码改动后必须执行 `gofmt`。
- 新增逻辑尽可能补充对应测试（store/handler/discord）。
- 数据结构变更要考虑 SQLite 迁移兼容。
- 面板功能变更应同步更新 `README.md` 的使用文档。

## 必跑检查（提交前）

至少执行以下命令：

```bash
go test ./...
go vet ./...
```

如果改动涉及页面交互，建议额外手动验证：

- `/admin/login` 登录流程
- `/` 安装登录流程
- `/portal` 安装入口展示
- `/admin` 的刷新、断开、拉黑动作

## Git 提交规范

推荐使用 Conventional Commits：

- `feat:` 新功能
- `fix:` 修复
- `docs:` 文档
- `refactor:` 重构
- `test:` 测试
- `chore:` 杂项

示例：

```text
feat(admin): add disconnect and blacklist management
fix(portal): block install for blacklisted guild
docs(readme): expand public deployment guide
```

## 分支开发与 PR 合并流程（CI/CD）

默认采用“功能分支 -> 签名提交 -> Pull Request -> CI 通过 -> 合并到 main”的流程。

### 1) 分支策略

- `main` 仅接受 PR 合并，不直接提交功能改动。
- 新需求从 `main` 拉出功能分支：`feat/*`、`fix/*`、`chore/*`。

### 2) 本地提交流程

- 完成改动后先执行：

```bash
go test ./...
go vet ./...
```

- 使用 GPG 智能卡进行签名提交（必须是 `Verified`）：

```bash
git commit -S -m "feat(...): ..."
```

### 3) 推送与 PR

- 推送功能分支到 `origin`。
- 发起指向 `main` 的 PR。
- PR 标题必须符合 Conventional Commits（如 `feat(admin): ...`）。

### 4) CI 必须通过

CI 在 `pull_request`、`push(main/tag)`、`merge_group` 触发，至少包含：

- `gofmt` 格式检查
- `go vet ./...`
- `go test ./...`

PR 仅在所有必需检查通过后可合并。

### 5) 合并与发布

- 通过评审后合并到 `main`（建议 Squash Merge，保持历史清晰）。
- 发布时打 `vX.Y.Z` 标签并推送。
- Tag 流水线负责构建并推送镜像到 `ghcr.io/freesialuo/dcportal`。

## 发布规范

- 默认发布分支：`main`
- 版本标签：`vX.Y.Z`（例如 `v0.1.5`）
- 推荐步骤：

```bash
go test ./...
go vet ./...
git push origin main
git tag vX.Y.Z
git push origin vX.Y.Z
```

镜像仓库：`ghcr.io/freesialuo/dcportal`

## 安全与风险控制

- 对授权与撤销逻辑的改动属于高风险变更，必须测试。
- 对数据库迁移的改动必须确保旧版本可平滑升级。
- 任何可能导致“误删安装记录”或“错误拉黑”的改动，应提供回退方案。

## 给自动化代理（AI Agent）的附加要求

- 不要执行破坏性 git 操作（如 `git reset --hard`）除非明确授权。
- 不要覆盖用户未要求修改的文件内容。
- 如果发现工作区存在与你任务无关的异常改动，应先暂停并确认。
- 输出结论时给出：
  - 改了哪些文件
  - 为什么这么改
  - 跑了哪些检查
  - 尚存哪些风险（如果有）

## 文档维护约定

以下情况必须同步更新 `README.md`：

- 新增/删除环境变量
- 新增后台操作入口或治理动作
- 路由、部署方式、镜像地址变化
- 版本发布流程变化

## 近期变更记录（Review Log）

以下为已合并到 `main` 的近期变更摘要（按 PR 顺序）：

### PR #2 `feat-admin-bot-management-ci-pr-gate`

- 管理后台新增 Bot 编辑能力（保留/清空敏感字段逻辑）。
- 新增 CI/PR 合并流程规范与校验项。
- 同步更新中英文 README 与 AGENTS 协作流程说明。

### PR #4 `chore-ci-pr-title-edited-trigger`

- CI `pull_request` 触发类型补充 `edited`，确保 PR 标题修改后重新校验 Conventional Commits。

### PR #5 `feat(portal): split bot install links`

- 引入 `install_links` 模型与迁移，支持“一个 Bot 对应多个安装链接（不同权限/Scopes/Redirect）”。
- Portal 安装入口改为按链接安装（`/install/{link_id}`）。
- 治理动作（黑名单、断开、退服）保持 Bot 维度。
- 补充 store/handler 测试，覆盖链接增删改查与回调流程。
- 后续评审修复已纳入：
  - 默认链接种子逻辑仅首次迁移执行；
  - 无 Redirect URI 场景默认链接禁用；
  - callback 增加 link/bot 启用状态复检；
  - 增补 links toggle/delete handler 测试。

### PR #6 `feat(ui): regroup admin bots and links into per-bot blocks`

- 管理后台 UI 改为按 Bot 分块展示，每个 Bot 下内聚显示 Links 列表与操作入口。
- 页面结构更贴合“Bot 主体 + Link 从属”的心智模型，减少跨区操作成本。

## Agent 交接速记（必读）

以下内容是给下一位 Agent 的高优先级上下文，建议开始开发前先阅读：

### 当前核心模型（2026-03）

- 安装入口已从“按 Bot”升级为“按 Install Link”：
  - 入口路由：`/install/{link_id}`
  - 数据关系：`bot (1) -> (N) install_links`
- 服务器治理动作仍按 `bot_id` 生效：
  - 黑名单（`guild_blacklist`）
  - 断开连接 / 退服
  - 刷新 Guild 信息

### 关键不变量（避免回归）

- `callback` 阶段必须复检 `bot.enabled` 与 `install_link.enabled`。
- `callback` 内记录安装与黑名单判断要以“当前查到的 link 归属 bot”作为准，不依赖旧 state 中的 bot 作为最终来源。
- 默认链接种子逻辑仅在 `install_links` 表首次引入时执行；不要在每次启动自动补回被管理员删除的最后一个 link。
- 当 Redirect URI 为空时，默认 link 应禁用，避免出现可点击但必失败的安装入口。

### Redirect URI 规则

- 授权跳转与 token exchange 必须使用同一份 redirect URI。
- install 时若 link.redirect_uri 为空，允许回退到 bot.redirect_uri；两者都为空时应拒绝安装（配置错误）。

### 数据迁移注意事项

- `internal/store/store.go` 包含向后兼容迁移逻辑；修改字段/表时先考虑旧库平滑升级。
- 涉及默认值策略时，优先保证“不破坏管理员显式操作结果”（例如管理员手动删除/禁用）。

### CI / PR 注意

- PR 标题必须符合 Conventional Commits（`feat(...)`, `fix(...)`, `chore(...)`...）。
- `.github/workflows/ci.yml` 已对 `pull_request.edited` 触发校验，改 PR 标题会重新跑校验。

### 建议优先人工验收路径

- `/admin`：Bot 编辑、Link 增删改、Link 启用/禁用
- `/portal`：同一 Bot 多链接展示与安装入口
- `/callback`：禁用 Bot/Link 后的回调拦截
- `/admin` 安装治理：Refresh / Revoke / Disconnect / Disconnect+Blacklist

## 当前分支进行中记录（feat/security-robust-ui-upgrade）

以下为尚未合并到 `main`、但已在功能分支落地的重要变更，供下一位 Agent 接续：

### 已完成提交（按时间顺序）

- `d409e91` `feat(security): enforce write-only discord secrets in admin`
  - 管理页改为 Secret 写入模型：`Client Secret` / `Bot Token` 不回显，仅支持覆盖或清空。
  - `internal/store` 新增 redacted admin 查询与 patch 更新能力，避免“读出现有 secret 再写回”的路径。
- `5ae3041` `fix(admin): validate and normalize bot/link inputs`
  - 增加 `permissions` 数字校验、`redirect_uri` 的 `http/https` 校验、`scopes` 归一化。
- `c9e80c6` `feat(ui): redesign admin and portal experience`
  - 重构 `admin/portal/login/result` 页面视觉与布局（含移动端适配）。
  - 更新中英文 README，补充写入式 Secret 行为说明。
- `c308f30` `docs(runbook): add ui manual verification checklist`
  - 新增手工验收清单：`docs/ui-manual-runbook.md`。
- `627623c` `test(ui): add frontend smoke and interaction coverage`
  - 新增轻量 UI 烟测：`internal/handler/ui_smoke_test.go`（模板渲染 + CSS 关键选择器 + 响应式规则）。

### 下一位 Agent 必知信息

- 管理页 Secret 规则现在是：
  - 留空：保持原值
  - 填新值：覆盖
  - `clear_bot_token=1`：清空 Bot Token
- UI 相关改动涉及文件集中在：
  - `web/templates/*.html`
  - `web/static/style.css`
  - `internal/handler/ui_smoke_test.go`
- 建议先跑自动检查再手动验收：
  - `go test ./...`
  - `go vet ./...`
  - `docs/ui-manual-runbook.md` 中的 10 项 UI 手工检查。

### 签名提交说明（给后续提交者）

- 当前分支最近提交均为签名提交（包含上述 5 个 commit）。
- 若 `git commit -S` 出现 `gpg: signing failed: Timeout`，默认视为“智能卡触摸未完成”：
  - 不要改用无签名提交；
  - 直接再次执行同一条 `git commit -S ...`，等待并完成触摸签名。
- 在受限环境下 `git log --show-signature` 可能因 keybox 访问受限无法完成信任校验；
  可用 `git cat-file -p <commit> | rg '^gpgsig '` 快速确认对象内含签名块。
