# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/dcportal ./cmd/dcportal/

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1000 dcportal
WORKDIR /app

# Copy binary and static assets
COPY --from=builder /bin/dcportal /app/dcportal
COPY web/ /app/web/
COPY configs/config.yaml /app/configs/config.yaml

# Create data directory
RUN mkdir -p /app/data && chown dcportal:dcportal /app/data

USER dcportal

EXPOSE 8080

# Environment variable configuration:
# DCPORTAL_PORT          - Server port (default: 8080)
# DCPORTAL_BASE_URL      - Public base URL (e.g. https://portal.example.com)
# DCPORTAL_ADMIN_TOKEN   - Admin authentication token (REQUIRED)
# DCPORTAL_INSTALL_TOKEN - Install portal token for distributed access (REQUIRED)
# DCPORTAL_DB_PATH       - SQLite database path (default: ./data/dcportal.db)

ENTRYPOINT ["/app/dcportal"]
CMD ["-config", "/app/configs/config.yaml"]
