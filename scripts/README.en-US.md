# DomainNest 1Panel HttpReq Patch

## Background

1Panel has two architecture modes:

- **Split Architecture**: Consists of `1panel-core` (Web UI API gateway, listens on TCP port) and `1panel-agent` (backend worker, listens on Unix socket `/etc/1panel/agent.sock`). Introduced in v1.10.28+.
- **Monolithic Architecture**: Single `1panel` binary handling everything (legacy mode).

The v1.10.34-lts source code uses split architecture, so the patch script can migrate monolithic installations to split architecture.

## Supported Versions

This patch adds HttpReq DNS provider support to 1Panel, supporting:

- **v1 LTS Split Architecture** (v1.10.28+, `1panel-core` + `1panel-agent`)
- **v1 LTS Monolithic Architecture** (upgraded legacy, `1panel` single binary)
- **v2 Split Architecture** (`1panel-core` + `1panel-agent`)

Requirements: Go 1.21+, git, patch

> **Note for v2 users**: DomainNest has native Technitium DNS support. This patch is only needed if you specifically require the HttpReq DNS provider.

## Features

- Automatic dependency checking and installation (git, patch, curl, Go)
- Go version auto-download and installation if system Go is too old
- Live build progress display with elapsed time
- Service file download for monolithic-to-split migration
- Idempotent (safe to run multiple times)
- Timeout protection on version detection commands

## Quick Install

```bash
# Download and extract the patch package
cd 1panel-patch
chmod +x patch-1panel.sh
sudo ./patch-1panel.sh
```

The script automatically performs the following steps:

**Split Architecture (v1 / v2):**
1. Detects 1Panel version and architecture (split/monolithic, v1/v2)
2. Selects the appropriate patch file
3. Clones source and verifies compatibility
4. Applies patch (idempotency check)
5. Builds and backs up original `1panel-agent` binary
6. Installs new binary and restarts service

**Monolithic Architecture v1 (v1.10.34-lts):**
1. Detects 1Panel version and architecture
2. Selects the appropriate patch file
3. Clones v1.10.34-lts source and verifies compatibility
4. Applies patch (idempotency check)
5. Builds `1panel-agent` (with HttpReq patch) and `1panel-core` (Web UI)
6. Backs up original monolithic binary as `1panel.backup.TIMESTAMP`
7. Installs both new binaries to `/usr/local/bin/`
8. Replaces `/usr/local/bin/1panel` with a symlink to `1panel-core`
9. Downloads `1panel-core.service` and `1panel-agent.service` from GitHub
10. Disables old `1panel.service`, enables and starts the two new services

## v1 Monolithic: Automatic Migration to Split Architecture

For monolithic v1 installations, the script performs the following migration:

- Builds both `1panel-agent` (with HttpReq patch) and `1panel-core`
- Backs up original monolithic binary as `1panel.backup.TIMESTAMP`
- Backs up `1panel.service` as `1panel.service.backup`
- Installs both binaries to `/usr/local/bin/`
- Replaces `/usr/local/bin/1panel` with a symlink to `1panel-core`
- Downloads official `1panel-core.service` and `1panel-agent.service` from GitHub
- Disables old `1panel.service`, enables and starts the two new services
- After migration, the system runs split architecture (same as official v1.10.34-lts release)

## Manual Steps

### v1 LTS Split Architecture

```bash
git clone --depth 1 --branch v1.10.34-lts https://github.com/1Panel-dev/1Panel.git /tmp/1panel
cd /tmp/1panel/agent
patch -p2 < /path/to/1panel-v1-httpreq.patch
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go
sudo cp 1panel-agent /usr/local/bin/1panel-agent
sudo systemctl restart 1panel-agent
```

### v2 Split Architecture

```bash
git clone --depth 1 --branch v2.1.13 https://github.com/1Panel-dev/1Panel.git /tmp/1panel
cd /tmp/1panel/agent
patch -p2 < /path/to/1panel-v2-httpreq.patch
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go
sudo cp 1panel-agent /usr/local/bin/1panel-agent
sudo systemctl restart 1panel-agent
```

> For monolithic v1 installations, we recommend using the automated script as the migration involves multiple binaries, symlinks, and service file changes that are difficult to perform manually.

## Configuration

In 1Panel, go to 「SSL Certificate」→「DNS Account」 and add:

- Type: **HttpReq**
- Endpoint: `https://your-domainnest-server/httpreq`
- Username: DomainNest AccessKeyID
- Password: DomainNest AccessKeySecret

## Rollback

To restore the original version:

```bash
chmod +x unpatch-1panel.sh
sudo ./unpatch-1panel.sh
```

The script automatically finds and restores the backup.

### Split Architecture Rollback

- Restores `1panel-agent` binary from backup
- Restarts service

### Migrated Monolithic Rollback

The script detects migration via backup files, then:

1. Stops `1panel-agent` and `1panel-core` services
2. Disables and removes split services and service files
3. Restores monolithic binary from backup (handles symlink correctly)
4. Restores `1panel.service` from backup
5. Re-enables and starts `1panel.service`

## Package Contents

The download zip contains:

- `1panel-v1-httpreq.patch` — Patch for v1 LTS
- `1panel-v2-httpreq.patch` — Patch for v2
- `patch-1panel.sh` — Install script (auto-selects language)
- `unpatch-1panel.sh` — Rollback script
- `README.md` — This file