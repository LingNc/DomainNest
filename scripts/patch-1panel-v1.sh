#!/bin/bash
set -e

# 1Panel v1 httpreq patch script
# Adds httpreq DNS provider support to 1Panel v1 for DomainNest integration

echo "=== 1Panel v1 httpreq Patch ==="
echo ""

# Check if 1Panel source is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <path-to-1panel-source>"
    echo ""
    echo "Example:"
    echo "  git clone https://github.com/1Panel-dev/1Panel.git --branch v1 --single-branch /tmp/1panel-v1"
    echo "  $0 /tmp/1panel-v1"
    exit 1
fi

PANEL_DIR="$1"
CLIENT_FILE="$PANEL_DIR/backend/utils/ssl/client.go"

if [ ! -f "$CLIENT_FILE" ]; then
    echo "Error: $CLIENT_FILE not found. Is this a valid 1Panel v1 source?"
    exit 1
fi

echo "[1/4] Backing up original file..."
cp "$CLIENT_FILE" "$CLIENT_FILE.bak"

echo "[2/4] Adding httpreq import..."
# Add httpreq import after existing dns provider imports
sed -i '/"github.com\/go-acme\/lego\/v4\/providers\/dns\/porkbun"/a\	"github.com/go-acme/lego/v4/providers/dns/httpreq"' "$CLIENT_FILE"

echo "[3/4] Adding httpreq case to getDNSProviderConfig..."
# Find the last case in the switch statement and add httpreq before the default
sed -i '/case PorkBun:/,/^		}/ {
    /^		}/ a\
\t\tcase HttpReq:\
\t\t\tconfig := httpreq.NewDefaultConfig()\
\t\t\tconfig.Endpoint = param.Endpoint\
\t\t\tconfig.Mode = "RAW"\
\t\t\tif param.Username != "" {\
\t\t\t\tconfig.Username = param.Username\
\t\t\t\tconfig.Password = param.Password\
\t\t\t}\
\t\t\tconfig.PropagationTimeout = propagationTimeout\
\t\t\tconfig.PollingInterval = pollingInterval\
\t\t\tp, err = httpreq.NewDNSProviderConfig(config)
}' "$CLIENT_FILE"

# Add HttpReq constant
sed -i '/PorkBun.*DnsType.*=.*"PorkBun"/a\\tHttpReq DnsType = "HttpReq"' "$CLIENT_FILE"

# Add Endpoint and Username/Password to DNSParam if not present
if ! grep -q 'Endpoint' "$CLIENT_FILE"; then
    sed -i '/Password.*string/a\\tEndpoint  string' "$CLIENT_FILE"
fi

echo "[4/4] Patch applied successfully!"
echo ""
echo "Next steps:"
echo "  1. Build 1Panel: cd $PANEL_DIR && go build -o 1panel ./cmd/server"
echo "  2. Stop 1Panel service: systemctl stop 1panel"
echo "  3. Replace binary: cp $PANEL_DIR/1panel /usr/local/bin/1panel"
echo "  4. Start 1Panel: systemctl start 1panel"
echo "  5. In 1Panel, create a DNS account with type 'HttpReq'"
echo "  6. Set Endpoint to: https://your-domainnest-server/httpreq"
echo "  7. Set Username to: your AccessKeyID"
echo "  8. Set Password to: your AccessKeySecret"
echo ""
echo "Backup saved at: $CLIENT_FILE.bak"