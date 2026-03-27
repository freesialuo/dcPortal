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
