# DomainNest 1Panel v1 httpreq Patch

## Compatibility

- 1Panel v1.x LTS (tested with v1.10.34-lts)
- Requirements: Go 1.21+, git, patch command

## Quick Install

```bash
cd 1panel-v1-patch
chmod +x patch-1panel-v1.sh
sudo ./patch-1panel-v1.sh
```

The script will:
1. Fetch the latest 1Panel v1 LTS tag
2. Clone source and verify compatibility
3. Apply patch (with idempotency check)
4. Build 1panel-agent
5. Backup original binary and replace
6. Restart service

## Manual Steps

```bash
git clone --depth 1 --branch v1.10.34-lts https://github.com/1Panel-dev/1Panel.git /tmp/1panel
cd /tmp/1panel/agent
patch -p2 < /path/to/1panel-v1-httpreq.patch
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go
sudo cp 1panel-agent /usr/local/bin/1panel-agent
sudo systemctl restart 1panel
```

## Configuration

In 1Panel SSL Certificates -> DNS Accounts:

- Type: **HttpReq**
- Endpoint: `https://your-domainnest-server/httpreq`
- Username: DomainNest AccessKeyID
- Password: DomainNest AccessKeySecret
