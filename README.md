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

1. 使用 Header 认证访问管理页面，例如：
   `curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/admin`
2. 添加 Bot：填写 Name、Client ID、Client Secret、Redirect URI（设为 `{BASE_URL}/callback`）、Permissions、Scopes
3. 用户访问 `http://localhost:8080/` 看到可安装的 Bot
4. 点击安装 → Discord OAuth2 授权 → 回调 → 安装成功

## 开发

```bash
make test        # 运行测试
make test-cover  # 带覆盖率
make vet         # 静态检查
make build       # 编译到 bin/dcportal
```

## CI/CD

- Push 到 `main` 或创建 `v*` tag 时自动运行测试并推送 Docker 镜像到 GHCR
- PR 仅运行测试
