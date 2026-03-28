# DCPortal

一个面向私有/半私有 Discord Bot 分发场景的授权门户。

English README: [README.en.md](./README.en.md)

DCPortal 通过“受控安装入口 + 管理后台 + 可审计安装记录 + 黑名单策略”，替代直接公开 Discord OAuth2 安装链接的方式，帮助团队更安全地管理 Bot 的安装与撤销。

- 镜像地址：`ghcr.io/freesialuo/dcportal:latest`
- 主要语言：Go
- 存储：SQLite
- 部署方式：二进制、Docker、Docker Compose

## 为什么使用 DCPortal

在很多团队里，Discord Bot 往往并不希望被任何人随意安装。

DCPortal 解决了这几个核心问题：

- 不直接暴露官方安装链接，而是由你控制安装入口。
- 安装页与管理页 token 分离，避免权限混用。
- 管理员可查看已安装服务器列表并主动执行治理动作。
- 支持断开、断开并拉黑、撤销 OAuth2 User Install 授权。
- 支持刷新服务器信息（名称、ID、成员数）用于运营盘点。

## 功能总览

### 访问与认证

- 安装页访问令牌（Install Token）。
- 管理页访问令牌（Admin Token）。
- 两类会话 Cookie 独立，不互相覆盖。

### Bot 管理

- 新增 Bot（Name / Client ID / Client Secret / Bot Token / Redirect URI / Scopes / Permissions）。
- 编辑已添加 Bot（支持修改名称、OAuth2 配置、权限与可选清空 Bot Token）。
- 按 Bot 管理多个安装链接（同一 Bot 可配置多组权限/Scope/Redirect URI）。
- 启用/禁用 Bot（控制是否在安装门户展示）。
- 删除 Bot（同时清理关联安装记录与黑名单记录）。

### OAuth2 安装流程

- 标准 OAuth2 authorization code flow。
- `state` 防 CSRF（单次使用，带过期）。
- 回调完成后记录安装信息。

### 服务器连接治理

- 刷新单条或全部安装记录（Guild Name / Member Count）。
- 撤销 User Install 授权（Revoke OAuth2 token）。
- 断开连接（删除记录并尝试让 Bot 退出服务器）。
- 断开并拉黑（后续安装到同一服务器将被拒绝）。

### 黑名单策略

- 黑名单按 `bot_id + guild_id` 维度生效。
- 被拉黑服务器再次安装时，会在回调阶段被拦截。
- 若配置了 Bot Token，系统会尝试让 Bot 立即离开该服务器。

## 页面与路由

| 路由 | 说明 |
|---|---|
| `/` | 安装入口登录页（Install Token） |
| `/portal` | Bot 安装选择页 |
| `/install/{id}` | 发起某个 Bot 的 OAuth2 安装 |
| `/callback` | Discord OAuth2 回调 |
| `/admin/login` | 管理后台登录页（Admin Token） |
| `/admin` | 管理后台首页 |

## 系统架构（简述）

- `cmd/dcportal/main.go`：程序入口、路由装配、中间件挂载。
- `internal/handler`：页面与业务流程。
- `internal/discord`：Discord API 封装（token exchange / revoke / guild fetch / leave guild）。
- `internal/store`：SQLite 数据访问与迁移。
- `internal/middleware`：安装/管理鉴权中间件。
- `web/templates` + `web/static`：前端页面模板与样式。

## 快速开始

### 1) 前置要求

- Go 1.25+
- 可访问 Discord OAuth2 API
- 一个 Discord Application（至少配置 OAuth2 Redirect URI）

### 2) 本地运行

```bash
# 1) 配置必须的 token
export DCPORTAL_ADMIN_TOKEN="replace-with-strong-admin-token"
export DCPORTAL_INSTALL_TOKEN="replace-with-strong-install-token"

# 2) 可选覆盖
export DCPORTAL_PORT="8080"
export DCPORTAL_BASE_URL="http://localhost:8080"
export DCPORTAL_DB_PATH="./data/dcportal.db"

# 3) 运行
make run
```

启动后访问：

- 安装登录页：`http://localhost:8080/`
- 管理登录页：`http://localhost:8080/admin/login`

### 3) Docker 运行

```bash
docker run -d \
  --name dcportal \
  -p 8080:8080 \
  -e DCPORTAL_ADMIN_TOKEN="replace-with-strong-admin-token" \
  -e DCPORTAL_INSTALL_TOKEN="replace-with-strong-install-token" \
  -e DCPORTAL_BASE_URL="https://portal.example.com" \
  -v dcportal-data:/app/data \
  ghcr.io/freesialuo/dcportal:latest
```

### 4) Docker Compose

```yaml
services:
  dcportal:
    image: ghcr.io/freesialuo/dcportal:latest
    container_name: dcportal
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      DCPORTAL_ADMIN_TOKEN: "replace-with-strong-admin-token"
      DCPORTAL_INSTALL_TOKEN: "replace-with-strong-install-token"
      DCPORTAL_BASE_URL: "https://portal.example.com"
      DCPORTAL_DB_PATH: "./data/dcportal.db"
    volumes:
      - dcportal-data:/app/data

volumes:
  dcportal-data:
```

## 配置说明

DCPortal 同时支持 YAML 配置文件与环境变量覆盖，环境变量优先级更高。

### `configs/config.yaml`

```yaml
server:
  port: 8080
  base_url: "http://localhost:8080"

admin:
  token: "change-me-to-a-secure-token"

install:
  token: "change-me-to-a-secure-token"

database:
  path: "./data/dcportal.db"
```

### 环境变量

| 变量 | 必需 | 默认值 | 说明 |
|---|---|---|---|
| `DCPORTAL_ADMIN_TOKEN` | 是 | 无 | 管理页认证 token |
| `DCPORTAL_INSTALL_TOKEN` | 是 | 无 | 安装页认证 token |
| `DCPORTAL_PORT` | 否 | `8080` | 服务端口 |
| `DCPORTAL_BASE_URL` | 否 | `http://localhost:8080` | 对外访问地址 |
| `DCPORTAL_DB_PATH` | 否 | `./data/dcportal.db` | SQLite 路径 |

> `ADMIN_TOKEN` 和 `INSTALL_TOKEN` 不能保留默认占位值，否则程序会拒绝启动。

## 管理后台操作手册

### 1) 新增 Bot

在 `/admin` 的 `Add Bot` 中填写：

- `Bot Name`：展示名。
- `Client ID`：Discord Application Client ID。
- `Client Secret`：OAuth2 code exchange 和 revoke 使用。
- `Bot Token`：用于获取服务器详情与执行退服（建议必填）。
- `Redirect URI`：必须与 Discord Developer Portal 配置完全一致。
- `Permissions`：权限位整数。
- `Scopes`：例如 `bot` 或 `bot applications.commands`。

### 2) 编辑 Bot

在 `/admin` 的 Bot 列表中点击 `Edit`，可修改已添加 Bot：

- `Bot Name` / `Client ID` / `Redirect URI` / `Permissions` / `Scopes`
- `Client Secret`：留空则保留原值
- `Bot Token`：留空则保留原值，也可勾选 `Clear Bot Token` 清空

### 3) 管理安装链接（每个 Bot 可多个）

在 `/admin` 的 `Install Links` 区域中，可为同一个 Bot 添加多条安装链接：

- `Link Name`：区分不同安装入口（如 Default / Lite / Full Access）。
- `Permissions` / `Scopes` / `Redirect URI`：按链接独立配置。
- 支持对链接执行编辑、启用/禁用、删除。
- 安装入口按链接路由：`/install/{link_id}`。

> 黑名单、断开连接、退服等治理动作仍按 Bot 维度执行。

### 4) 安装治理动作

对于每条安装记录，可执行：

- `Refresh`：刷新该服务器名称与成员数。
- `Revoke OAuth2`：撤销用户授权 token。
- `Disconnect`：删除连接记录，并尝试让 Bot 退出服务器。
- `Disconnect + Blacklist`：断开并拉黑，后续再次安装会被拒绝。

批量动作：

- `Refresh All Guild Info`：刷新全部安装记录。

## 安全建议

- 使用高强度随机 token（Admin/Install 至少 32 位）。
- 生产环境务必使用 HTTPS 与反向代理（Nginx/Caddy/Traefik）。
- 对 `/admin` 建议增加 IP 白名单或二次访问控制。
- 不要把 `Client Secret`、`Bot Token` 提交到仓库。
- 定期备份 SQLite 数据文件。

## 反向代理建议

确保将真实外部 URL 与 `DCPORTAL_BASE_URL`、Discord 应用中的 Redirect URI 保持一致。

常见失败原因：

- 协议不一致（`http` vs `https`）。
- 端口不一致。
- 路径不一致（尤其 `/callback`）。

## 数据与持久化

默认数据库位置：`./data/dcportal.db`

生产部署建议：

- Docker 使用 volume 挂载 `/app/data`。
- 升级前备份数据库文件。
- 数据库与应用分盘，避免容器销毁导致数据丢失。

## 开发与测试

```bash
make test        # go test ./... -v
make test-cover  # go test ./... -cover
make vet         # go vet ./...
make build       # 生成 bin/dcportal
```

## 发布流程（建议）

1. 在 `main` 分支完成变更并通过测试。
2. 创建带语义版本号的 tag（例如 `v0.1.5`）。
3. CI 在 tag 流水线中构建并发布镜像到 GHCR。
4. 部署端更新镜像版本或使用 `latest`。

## 升级说明

如果你从较旧版本升级到当前版本：

- 程序启动时会自动执行 SQLite 迁移（新增字段/表）。
- 建议升级前先备份数据库。
- 升级后在管理页为已有 Bot 补齐 `Bot Token`，否则“刷新服务器信息/退服”会受限。

## 常见问题（FAQ）

### Q1: 登录后仍被重定向到登录页

- 检查 token 是否输错（Admin 与 Install 不通用）。
- 检查浏览器是否拦截 Cookie。

### Q2: OAuth2 回调失败

- 重点核对 Redirect URI（协议、域名、端口、路径必须完全一致）。
- 核对 `DCPORTAL_BASE_URL` 是否是用户实际访问地址。

### Q3: 为什么成员数没有刷新

- 对应 Bot 未配置 `Bot Token`。
- Bot 在目标服务器缺少可见权限或已不在服务器内。

### Q4: 断开后为什么 Bot 还在服务器里

- 常见原因是 `Bot Token` 缺失或失效。
- 管理页会完成“记录断开”，但退服动作可能失败。

## 许可证

本项目采用 `AGPL-3.0-only` 许可证，详见 [LICENSE](./LICENSE)。

如果你以网络服务方式提供二次开发版本，根据 AGPL 要求，需要向用户提供对应源代码。

## 致谢

如果这个项目帮到了你，欢迎 Star、提 Issue、或提交 PR 一起完善。
