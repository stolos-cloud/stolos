#!/bin/bash
#set -euo pipefail

HOST="root@192.168.2.50"     # Proxmox node
CLI_HOST="root@10.0.21.32" # Remote LXC or SSH target
STORAGE="sas-sdb"        # Storage pool
SIZE="10"                  # Disk size (e.g., 32G)
RESET=1                    # 1 = reset disks, 0 = skip
DEBUG=                     # Set non-empty to enable Delve
RUN_LOCAL=1                # 1 = run CLI locally, 0 = run remotely
SKIP_CLI=1                 # 1 = only reset/reboot, skip compile/run

# Define VMIDs here (comma-separated)
VMIDS="204"

# Compile & transfer CLI
if [[ "$SKIP_CLI" -eq 0 ]]; then
    if [[ "$RUN_LOCAL" -eq 1 ]]; then
        echo "Compiling locally..."
        CGO_ENABLED=0 go build -gcflags="all=-N -l" \
          -ldflags="-extldflags=\"-static\" \
            -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientId=$GCP_CLIENT_ID \
            -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientSecret=$GCP_CLIENT_SECRET" \
          -o out/bootstrap ./cmd/bootstrap
        echo "Build complete (local)."
    else
        echo "Compiling..."
        CGO_ENABLED=0 go build -gcflags="all=-N -l" \
          -ldflags="-extldflags=\"-static\" \
            -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientId=$GCP_CLIENT_ID \
            -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientSecret=$GCP_CLIENT_SECRET" \
          -o out/bootstrap ./cmd/bootstrap
        echo "Done. Sending file to remote..."
        scp ./out/bootstrap $CLI_HOST:bootstrap
    fi
fi

# Convert to space-separated list
IFS=',' read -r -a VMID_ARRAY <<< "$VMIDS"

for VMID in "${VMID_ARRAY[@]}"; do
    echo "===== Processing VM $VMID ====="

    ssh "$HOST" bash -s <<EOF
set -euo pipefail

echo ">>> Stopping VM $VMID..."
qm stop "$VMID"

if [[ "$RESET" -eq 1 ]]; then
    echo ">>> Detecting first disk..."
    FIRST_DISK=\$(qm config "$VMID" | grep -E '^(virtio0|scsi0|sata0|ide0):' | head -n1 | cut -d: -f1)

    if [[ -z "\$FIRST_DISK" ]]; then
        echo "No primary disk found. Adding new disk on scsi0..."
        qm set "$VMID" --scsi0 "$STORAGE:$SIZE"
    else
        echo ">>> Unlinking and deleting disk \$FIRST_DISK from VM $VMID..."
        qm disk unlink "$VMID" --idlist "\$FIRST_DISK" --force

        echo ">>> Adding new disk ($SIZE on $STORAGE) to \$FIRST_DISK..."
        qm set "$VMID" --"\$FIRST_DISK" "$STORAGE:$SIZE"
    fi
fi

echo ">>> Starting VM $VMID..."
qm start "$VMID"

echo ">>> VM $VMID done."
EOF

done

# Skip CLI execution if requested
if [[ "$SKIP_CLI" -eq 1 ]]; then
    echo "Skipping CLI execution (SKIP_CLI=1)."
    exit 0
fi

# Run CLI
if [[ "$RUN_LOCAL" -eq 1 ]]; then
    rm -f talos-bootstrap-state.json || true
    if [[ -n "$DEBUG" ]]; then
        go/bin/dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./out/bootstrap
    else
        ./out/bootstrap
    fi
else
    ssh $CLI_HOST rm -f talos-bootstrap-state.json || true
    if [[ -n "$DEBUG" ]]; then
        ssh -t $CLI_HOST go/bin/dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./bootstrap
    else
        ssh -t $CLI_HOST ./bootstrap
    fi
fi
