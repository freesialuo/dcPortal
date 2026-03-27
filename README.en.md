# DCPortal

An authorization portal for private or semi-private Discord Bot distribution.

DCPortal replaces directly exposing Discord OAuth2 install links with a controlled flow:

- gated install entry
- admin management console
- auditable install records
- blacklist policy for guild-level control

- Docker image: `ghcr.io/freesialuo/dcportal:latest`
- Language: Go
- Storage: SQLite
- Deployment: binary, Docker, Docker Compose

## Why DCPortal

Many teams do not want anyone to freely install internal bots via public links.

DCPortal helps by providing:

- controlled install entry instead of direct public OAuth2 links
- separated install/admin tokens to avoid privilege overlap
- visible install records and active governance operations
- disconnect, disconnect+blacklist, OAuth2 user authorization revocation
- guild info refresh (name, ID, member count) for operations

## Feature Overview

### Access and Authentication

- Install token for install portal access
- Admin token for admin console access
- Independent session cookies for install/admin contexts

### Bot Management

- Add bot (Name / Client ID / Client Secret / Bot Token / Redirect URI / Scopes / Permissions)
- Enable/disable bot visibility on the install portal
- Delete bot and related install/blacklist records

### OAuth2 Install Flow

- Standard OAuth2 authorization code flow
- `state`-based CSRF protection (single-use with expiration)
- Install record persistence after callback

### Guild Governance

- Refresh one or all install records (guild name / member count)
- Revoke User Install authorization (OAuth2 token revoke)
- Disconnect link (remove record and attempt bot leave)
- Disconnect + blacklist (future installs to same guild are blocked)

### Blacklist Policy

- Scope: `bot_id + guild_id`
- Blocked guild installs are rejected during callback
- If Bot Token is configured, bot leave is attempted immediately

## Routes

| Route | Description |
|---|---|
| `/` | Install login page (Install Token) |
| `/portal` | Bot selection page |
| `/install/{id}` | Start OAuth2 install for a bot |
| `/callback` | Discord OAuth2 callback |
| `/admin/login` | Admin login page (Admin Token) |
| `/admin` | Admin dashboard |

## Architecture (Short)

- `cmd/dcportal/main.go`: app entry, routing, middleware wiring
- `internal/handler`: web handlers and flow orchestration
- `internal/discord`: Discord API client wrapper
- `internal/store`: SQLite access and migration
- `internal/middleware`: auth middleware
- `web/templates` + `web/static`: UI templates and static assets

## Quick Start

### 1) Requirements

- Go 1.25+
- Discord OAuth2 API reachable
- A Discord Application with OAuth2 Redirect URI configured

### 2) Run Locally

```bash
export DCPORTAL_ADMIN_TOKEN="replace-with-strong-admin-token"
export DCPORTAL_INSTALL_TOKEN="replace-with-strong-install-token"

# optional overrides
export DCPORTAL_PORT="8080"
export DCPORTAL_BASE_URL="http://localhost:8080"
export DCPORTAL_DB_PATH="./data/dcportal.db"

make run
```

Then open:

- Install login: `http://localhost:8080/`
- Admin login: `http://localhost:8080/admin/login`

### 3) Run with Docker

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

## Configuration

DCPortal supports YAML config with environment overrides (env wins).

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

### Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DCPORTAL_ADMIN_TOKEN` | Yes | N/A | Admin auth token |
| `DCPORTAL_INSTALL_TOKEN` | Yes | N/A | Install auth token |
| `DCPORTAL_PORT` | No | `8080` | Server port |
| `DCPORTAL_BASE_URL` | No | `http://localhost:8080` | Public base URL |
| `DCPORTAL_DB_PATH` | No | `./data/dcportal.db` | SQLite path |

`ADMIN_TOKEN` and `INSTALL_TOKEN` must not stay as default placeholders.

## Admin Operations

### Add Bot

Configure in `/admin`:

- Bot Name
- Client ID
- Client Secret
- Bot Token (recommended for guild refresh and leave)
- Redirect URI (must exactly match Discord Developer Portal)
- Permissions
- Scopes (`bot` or `bot applications.commands`)

### Install Governance

Per install record:

- `Refresh`
- `Revoke OAuth2`
- `Disconnect`
- `Disconnect + Blacklist`

Batch action:

- `Refresh All Guild Info`

## Security Recommendations

- Use strong random tokens (at least 32 chars)
- Use HTTPS in production (with reverse proxy)
- Add additional protection to `/admin` (IP allowlist, etc.)
- Never commit secrets (`Client Secret`, `Bot Token`)
- Back up SQLite DB regularly

## Reverse Proxy Notes

Keep these fully consistent:

- external URL
- `DCPORTAL_BASE_URL`
- Discord Redirect URI

Typical mismatch issues:

- protocol mismatch (`http` vs `https`)
- port mismatch
- path mismatch (`/callback`)

## Persistence

Default DB path: `./data/dcportal.db`

Production tips:

- mount `/app/data` as volume in Docker
- backup before upgrading
- separate app and data lifecycle

## Development

```bash
make test
make test-cover
make vet
make build
```

## Release (Recommended)

1. Merge to `main`
2. Tag semantic version (e.g. `v0.1.5`)
3. CI builds/publishes GHCR image
4. Deploy by pinned tag or `latest`

## Upgrade Notes

- DB migrations run automatically at startup
- Back up DB before upgrade
- fill Bot Token for existing bots after upgrade to enable refresh/leave features

## FAQ

### Redirect loop after login

- check token type (admin vs install)
- check browser cookie policies

### OAuth2 callback failure

- verify Redirect URI exact match
- verify `DCPORTAL_BASE_URL` matches real access URL

### Member count not refreshing

- missing/invalid Bot Token
- bot not in guild or lacks visibility/permission

### Bot did not leave after disconnect

- Bot Token missing/invalid
- record may be removed while leave call fails

## License

No LICENSE file is currently included. For open-source release, consider adding MIT or Apache-2.0.

## Contributing

Issues and PRs are welcome.
