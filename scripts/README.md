# DomainNest Scripts

## 1Panel v1 httpreq Patch

1Panel v1 does not include the httpreq DNS provider. This patch adds it, allowing 1Panel v1 to use DomainNest as a DNS provider for ACME certificate challenges.

### Prerequisites

- Go 1.21+ installed
- Git installed
- 1Panel v1 source code

### Usage

```bash
# Clone 1Panel v1 source
git clone https://github.com/1Panel-dev/1Panel.git --branch v1 --single-branch /tmp/1panel-v1

# Run the patch script
./patch-1panel-v1.sh /tmp/1panel-v1

# Build and install
cd /tmp/1panel-v1
go build -o 1panel ./cmd/server
sudo systemctl stop 1panel
sudo cp 1panel /usr/local/bin/1panel
sudo systemctl start 1panel
```

### Configuration in 1Panel

1. Go to **Settings** -> **DNS Accounts**
2. Click **Add DNS Account**
3. Select **HttpReq** as the type
4. Fill in:
   - **Endpoint**: `https://your-domainnest-server/httpreq`
   - **Username**: Your DomainNest AccessKeyID
   - **Password**: Your DomainNest AccessKeySecret
5. Save

### Alternative: 1Panel v2

1Panel v2 (v2.1.2+) supports **Technitium** and **AcmeDNS** natively, which are also compatible with DomainNest:

- **Technitium**: Select "Technitium", set BASE URL to `https://your-server/technitium`, Token to your RAM token
- **AcmeDNS**: Select "AcmeDNS", set API BASE to `https://your-server/acmedns`

No patching needed for v2.