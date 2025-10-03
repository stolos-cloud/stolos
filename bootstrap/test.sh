#!/bin/bash

#set -euo pipefail


HOST="root@10.0.21.51"     # e.g. root@proxmox-node
CLI_HOST="root@10.0.21.32"
STORAGE="RBD_Common"  # e.g. local-lvm
SIZE="20"     # e.g. 32G
RESET=1
DEBUG=

# Define VMIDs here (comma-separated)
VMIDS="200003,200005,200006,200007"

echo "Compiling..."
CGO_ENABLED=0 go build -gcflags="all=-N -l" -ldflags="-extldflags=\"-static\" -X github.com/stolos-cloud/stolos-bootstrap/pkg/github.GithubClientId=$GH_CLIENT_ID -X github.com/stolos-cloud/stolos-bootstrap/pkg/github.GithubClientSecret=$GH_CLIENT_SECRET -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientId=$GCP_CLIENT_ID -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientSecret=$GCP_CLIENT_SECRET" -o out/bootstrap ./cmd/bootstrap
echo "Done. Sending file to remote"
scp ./out/bootstrap $CLI_HOST:bootstrap

# Convert to space-separated list
IFS=',' read -r -a VMID_ARRAY <<< "$VMIDS"

for VMID in "${VMID_ARRAY[@]}"; do
    echo "===== Processing VM $VMID ====="
    
    ssh "$HOST" bash -s <<EOF
set -euo pipefail

echo ">>> Stopping VM $VMID..."
qm stop "$VMID"

if [[ -n "$RESET" ]]; then
echo ">>> Detecting first disk..."
FIRST_DISK=\$(qm config "$VMID" | grep -E '^(virtio0|scsi0|sata0|ide0):' | head -n1 | cut -d: -f1)

if [[ -z "\$FIRST_DISK" ]]; then
    echo "No primary disk found (virtio0/scsi0/sata0/ide0)."
    exit 1
fi

echo ">>> Unlinking and deleting disk \$FIRST_DISK from VM $VMID..."
qm disk unlink "$VMID" --idlist "\$FIRST_DISK" --force

echo ">>> Adding new disk ($SIZE on $STORAGE)..."
qm set "$VMID" --"\$FIRST_DISK" "$STORAGE:$SIZE"
fi

echo ">>> Starting VM $VMID..."
qm start "$VMID"

echo ">>> VM $VMID done."
EOF

done

ssh $CLI_HOST rm talos-bootstrap-state.json || true
if [[ -n "$DEBUG" ]]; then
    ssh -t $CLI_HOST go/bin/dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./bootstrap
else
    ssh -t $CLI_HOST ./bootstrap
fi
