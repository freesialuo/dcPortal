# DCPortal

Discord Bot 授权管理门户 — 通过自定义链接控制私有 Bot 的安装授权，替代 Discord 官方公开链接。

## 功能

- 管理员通过 Token 认证管理 Bot 列表（添加 / 启用 / 禁用 / 删除）
- 完整 Discord OAuth2 授权码流程（state 验证 → code 交换 → guild 记录）
- 仅展示已启用的 Bot 给用户安装
- 安装记录跟踪（Guild ID / Name / 时间）
- 深色主题 Web UI

## 环境变量

| 变量 | 必须 | 默认值 | 说明 |
|------|------|--------|------|
| `DCPORTAL_ADMIN_TOKEN` | ✅ | — | 管理员认证 Token |
| `DCPORTAL_PORT` | | `8080` | 监听端口 |
| `DCPORTAL_BASE_URL` | | `http://localhost:8080` | 公开访问 URL |
| `DCPORTAL_DB_PATH` | | `./data/dcportal.db` | SQLite 数据库路径 |

## Quick Start

### 本地运行

```bash
# 设置 Admin Token（必须）
export DCPORTAL_ADMIN_TOKEN="your-secure-token"

# 编辑配置
vim configs/config.yaml

# 运行
make run
```

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e DCPORTAL_ADMIN_TOKEN="your-secure-token" \
  -e DCPORTAL_BASE_URL="https://portal.example.com" \
  -v dcportal-data:/app/data \
  ghcr.io/YOUR_USER/dcportal:latest
```

### Docker Compose

```yaml
services:
  dcportal:
    image: ghcr.io/YOUR_USER/dcportal:latest
    ports:
      - "8080:8080"
    environment:
      DCPORTAL_ADMIN_TOKEN: "your-secure-token"
      DCPORTAL_BASE_URL: "https://portal.example.com"
    volumes:
      - dcportal-data:/app/data

volumes:
  dcportal-data:
```

## 使用流程

1. 浏览器访问 `http://localhost:8080/`，先输入 `ADMIN token` 完成登录。
2. 登录后访问管理页 `http://localhost:8080/admin` 添加 Bot：填写 Name、Client ID、Client Secret、Redirect URI（设为 `{BASE_URL}/callback`）、Permissions、Scopes。
3. 登录后访问安装页 `http://localhost:8080/portal` 查看可安装 Bot。
4. 点击安装 → Discord OAuth2 授权 → 回调（`/callback`）→ 安装成功。

CLI 方式仍支持 Header 认证，例如：
`curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/admin`

## 面板详细使用指南

### 1) 首次登录与会话

1. 打开首页 `/`，输入 `ADMIN token` 登录。
2. 登录成功后会写入 `admin_token` Cookie，用于后续页面访问鉴权。
3. 点击页面底部 `Logout` 可立即清除会话。

### 2) 新增 Bot（Admin 页面）

在 `/admin` 的 `Add Bot` 表单中：

- `Bot Name`：仅用于面板显示。
- `Client ID`：Discord Application 的 Client ID（必填）。
- `Client Secret`：Discord Application 的 Client Secret（必填，敏感信息）。
- `Redirect URI`：必须与 Discord Developer Portal 中配置一致，通常为 `{BASE_URL}/callback`。
- `Permissions`：Discord 权限位整数（十进制）。
- `Scopes`：常用为 `bot`，如果需要 Slash 命令可用 `bot applications.commands`。

保存后 Bot 默认为 `Enabled`，会出现在 `/portal` 列表。

### 3) 使用官方权限计算器（已接入）

在 `Permissions` 字段下方点击 `Open Official Calculator`：

1. 若 `Client ID` 有值，会跳转到 Discord 官方开发者后台该应用的 `OAuth2 URL Generator`。
2. 若 `Client ID` 为空，会跳转到 Discord 应用列表页，先选择应用再进入 URL Generator。
3. 在官方页面勾选所需权限后，复制生成的整数权限值，粘贴回 DCPortal 的 `Permissions` 字段。

说明：官方计算器在 Discord Developer Portal 中运行，需要你已登录 Discord 开发者账号。

### 4) 安装与回调验证

1. 在 `/portal` 点击 `Install Bot`。
2. Discord 授权成功后会回调 `/callback`。
3. 成功页面会显示 Bot、Guild 信息；同时安装记录写入 Admin 的 `Install History`。

### 5) Bot 维护操作

- `Enable/Disable`：控制 Bot 是否在 `/portal` 对外可见。
- `Delete`：删除 Bot 配置和对应安装记录（不可恢复）。

### 6) 常见问题排查

- 登录后仍提示未授权：
  - 检查 `DCPORTAL_ADMIN_TOKEN` 是否与输入值一致。
  - 检查浏览器是否禁用了 Cookie。
- Discord 回调失败：
  - 核对 `Redirect URI` 与 Discord 应用后台完全一致（包含协议、域名、端口、路径）。
  - 确认 `BASE_URL` 配置与实际访问地址一致。
- Bot 没出现在 `/portal`：
  - 确认该 Bot 状态为 `Enabled`。

## 开发

```bash
make test        # 运行测试
make test-cover  # 带覆盖率
make vet         # 静态检查
make build       # 编译到 bin/dcportal
```

## CI/CD

- Push 到 `main`：仅运行测试（`vet` + `test`）
- Push `v*` tag：运行测试并发布 Docker 镜像到 GHCR
- PR：仅运行测试

推荐发布流程：

1. 提交代码并推送到 `main`
2. 确认 `main` 的 CI 测试通过
3. 创建并推送版本 tag（如 `v0.1.3`）触发发布
