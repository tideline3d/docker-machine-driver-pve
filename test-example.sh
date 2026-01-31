#!/bin/bash
# Example test script for the PVE driver with network configuration
# This script runs inside the Docker test container

set -e

# Configuration - EDIT THESE VALUES
PVE_URL="${PVE_URL:-https://10.42.1.93:8006}"
PVE_TOKEN_ID="${PVE_TOKEN_ID:-root@pam!packer}"
PVE_TOKEN_SECRET="${PVE_TOKEN_SECRET:-cbc51ad1-59c1-4c43-b1e4-0aec353a3411}"
PVE_RESOURCE_POOL="${PVE_RESOURCE_POOL:-k3s}"
PVE_TEMPLATE="${PVE_TEMPLATE:-9000}"
PVE_ISO_DEVICE="${PVE_ISO_DEVICE:-scsi1}"
PVE_NETWORK_INTERFACE="${PVE_NETWORK_INTERFACE:-net0}"

# Network configuration
PVE_NETWORK_BRIDGE_0="${PVE_NETWORK_BRIDGE_0:-vmbr0}"
PVE_NETWORK_VLAN_0="${PVE_NETWORK_VLAN_0:-12}"
PVE_NETWORK_BRIDGE_1="${PVE_NETWORK_BRIDGE_1:-}"
PVE_NETWORK_VLAN_1="${PVE_NETWORK_VLAN_1:-}"

# VM name
VM_NAME="${VM_NAME:-test-network-vm-$(date +%s)}"

echo "=== PVE Driver Test with Network Configuration ==="
echo ""
echo "Configuration:"
echo "  PVE URL: $PVE_URL"
echo "  Token ID: $PVE_TOKEN_ID"
echo "  Resource Pool: $PVE_RESOURCE_POOL"
echo "  Template: $PVE_TEMPLATE"
echo "  VM Name: $VM_NAME"
echo ""
echo "Network Configuration:"
echo "  Configure Network: true"
echo "  Bridge 0: $PVE_NETWORK_BRIDGE_0"
echo "  VLAN 0: $PVE_NETWORK_VLAN_0"
if [ -n "$PVE_NETWORK_BRIDGE_1" ]; then
  echo "  Bridge 1: $PVE_NETWORK_BRIDGE_1"
  echo "  VLAN 1: $PVE_NETWORK_VLAN_1"
fi
echo ""

# Check if required variables are set
if [ "$PVE_TOKEN_SECRET" = "YOUR_SECRET_HERE" ]; then
  echo "ERROR: Please set PVE_TOKEN_SECRET environment variable"
  echo ""
  echo "You can either:"
  echo "  1. Edit this file and change YOUR_SECRET_HERE"
  echo "  2. Set environment variables before running:"
  echo ""
  echo "     export PVE_TOKEN_SECRET=your-actual-secret"
  echo "     export PVE_URL=https://your-proxmox:8006"
  echo "     ./test-example.sh"
  exit 1
fi

echo "Creating VM with network configuration..."
echo ""

# Build the docker-machine command
# Note: Using array to properly handle arguments with special characters
CMD=(docker-machine create
  --driver pve
  --pve-url "$PVE_URL"
  --pve-token-id "$PVE_TOKEN_ID"
  --pve-token-secret "$PVE_TOKEN_SECRET"
  --pve-resource-pool "$PVE_RESOURCE_POOL"
  --pve-template "$PVE_TEMPLATE"
  --pve-iso-device "$PVE_ISO_DEVICE"
  --pve-network-interface "$PVE_NETWORK_INTERFACE"
  --pve-configure-network
  --pve-network-bridge-0 "$PVE_NETWORK_BRIDGE_0"
  --pve-network-vlan-0 "$PVE_NETWORK_VLAN_0"
)

# Add second NIC if configured
if [ -n "$PVE_NETWORK_BRIDGE_1" ]; then
  CMD+=(--pve-network-bridge-1 "$PVE_NETWORK_BRIDGE_1")
  if [ -n "$PVE_NETWORK_VLAN_1" ]; then
    CMD+=(--pve-network-vlan-1 "$PVE_NETWORK_VLAN_1")
  fi
fi

# Add VM name
CMD+=("$VM_NAME")

# Print the command for debugging
echo "Running command:"
echo "${CMD[@]}"
echo ""

# Execute the command
"${CMD[@]}"

echo ""
echo "=== VM Created Successfully! ==="
echo ""
echo "To verify network configuration, SSH to your Proxmox host and run:"
echo "  qm config <VM_ID> | grep net"
echo ""
echo "To clean up (remove the VM):"
echo "  docker-machine rm $VM_NAME"
echo ""
