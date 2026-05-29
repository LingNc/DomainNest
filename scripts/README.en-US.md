# DomainNest 1Panel HttpReq Patch

## Supported Versions

This patch adds HttpReq DNS provider support to 1Panel, supporting:

- **v1 LTS Split Architecture** (v1.10.28+, `1panel-core` + `1panel-agent`)
- **v1 LTS Monolithic Architecture** (upgraded legacy, `1panel` single binary)
- **v2 Split Architecture** (`1panel-core` + `1panel-agent`)

Requirements: Go 1.21+, git, patch

## Quick Install

```bash
# Download and extract the patch package
cd 1panel-patch
chmod +x patch-1panel.sh
sudo ./patch-1panel.sh
```

The script automatically:
1. Detects 1Panel version and architecture (split/monolithic, v1/v2)
2. Selects the appropriate patch file
3. Clones source and verifies compatibility
4. Applies patch (idempotency check)
5. Builds and backs up original binary
6. Restarts service

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

The script automatically finds and restores the backup. Backup files are located in the same directory as the original binary, format: `*.backup.YYYYMMDDhhmmss`.