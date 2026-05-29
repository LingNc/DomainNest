# DomainNest 1Panel HttpReq Patch

Adds HttpReq DNS provider support to 1Panel.

## Supported Versions

- **v1 LTS Split Architecture** (v1.10.28+, `1panel-core` + `1panel-agent`)
- **v1 LTS Monolithic Architecture** (upgraded legacy, `1panel` single binary)
- **v2 Split Architecture** (`1panel-core` + `1panel-agent`)

Requirements: Go 1.21+, git, patch, npm (optional, for frontend patch build)

> **Note for v2 users**: DomainNest has native Technitium DNS support. This patch is only needed if you specifically require the HttpReq DNS provider.

## Quick Install

```bash
cd 1panel-patch
chmod +x patch-1panel.sh
sudo ./patch-1panel.sh
```

## What the Script Does

- Detects 1Panel version and architecture type (split/monolithic, v1/v2)
- Selects the appropriate patch file and verifies compatibility
- Builds new `1panel-agent` binary with HttpReq support
- Handles monolithic-to-split migration automatically (binary replacement, symlink, service files)
- Backs up originals; idempotent (safe to run multiple times)
- Rollback script restores original version (detects migration state, full restore)

## Rollback

```bash
chmod +x unpatch-1panel.sh
sudo ./unpatch-1panel.sh
```

- **Split architecture rollback**: restores `1panel-agent` binary, restarts service
- **Monolithic rollback**: stops/disables split services, restores monolithic binary, `1panel.service`, and symlink from backup

## Package Contents

- `1panel-v1-httpreq.patch` — Backend patch for v1 LTS
- `1panel-v2-httpreq.patch` — Backend patch for v2
- `1panel-httpreq-frontend.patch` — Frontend patch (adds HttpReq DNS provider option)
- `patch-1panel.sh` — Install script
- `unpatch-1panel.sh` — Rollback script