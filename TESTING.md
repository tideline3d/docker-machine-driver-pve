# Testing Guide - Docker-based Isolated Testing

This guide shows you how to test the PVE driver completely within Docker, without polluting your development machine.

## Prerequisites

- Docker installed and running
- Task (taskfile) installed
- Access to a Proxmox VE server

## Quick Start

### Option 1: Quick Command Test (Recommended for verification)

Test individual commands without entering the container:

```bash
# Check docker-machine version
./test-quick.sh docker-machine --version

# Check driver version
./test-quick.sh docker-machine-driver-pve --version

# View driver help
./test-quick.sh docker-machine create --driver pve --help
```

### Option 2: Interactive Container (Recommended for development)

Enter an interactive shell in the test container:

```bash
# Build and enter the test container
./test-docker.sh

# Inside the container, you can run commands like:
docker-machine --version
docker-machine-driver-pve --version
docker-machine create --driver pve --help
```

### Option 3: Full Integration Test

Run a complete test that creates a VM with network configuration:

```bash
# Set your credentials as environment variables
export PVE_URL="https://your-proxmox:8006"
export PVE_TOKEN_ID="root@pam!rancher"
export PVE_TOKEN_SECRET="your-secret-here"
export PVE_RESOURCE_POOL="your-pool"
export PVE_TEMPLATE="9000"

# Configure network settings
export PVE_NETWORK_BRIDGE_0="vmbr0"
export PVE_NETWORK_VLAN_0="12"

# Run the test
./test-quick.sh ./test-example.sh
```

Or edit `test-example.sh` with your values and run it interactively:

```bash
./test-docker.sh
# Inside container:
./test-example.sh
```

## Detailed Usage

### Method 1: Quick Testing (`test-quick.sh`)

Best for running one-off commands:

```bash
# Basic syntax
./test-quick.sh <command>

# Examples
./test-quick.sh docker-machine --version
./test-quick.sh docker-machine ls
./test-quick.sh docker-machine create --driver pve --help
```

**Pros:**
- Quick, no need to enter container
- Perfect for CI/CD pipelines
- Environment variables are passed through

**Cons:**
- Less interactive
- Need to rebuild for each command

### Method 2: Interactive Container (`test-docker.sh`)

Best for development and debugging:

```bash
./test-docker.sh
```

This will:
1. Build the driver binary
2. Build the test Docker image
3. Drop you into an interactive shell

Inside the container:

```bash
# Your driver is available at /usr/local/bin/docker-machine-driver-pve
# Your code is mounted at /workspace

# Test the driver
# IMPORTANT: Use single quotes around token-id to avoid bash history expansion
docker-machine create \
  --driver pve \
  --pve-url "https://10.42.1.93:8006" \
  --pve-token-id 'root@pam!packer' \
  --pve-token-secret "cbc51ad1-59c1-4c43-b1e4-0aec353a3411" \
  --pve-insecure-tls \
  --pve-resource-pool "k3s" \
  --pve-template "5959" \
  --pve-iso-device "scsi1" \
  --pve-network-interface "net0" \
  --pve-configure-network \
  --pve-network-bridge-0 "vmbr0" \
  --pve-network-vlan-0 "12" \
  --pve-network-bridge-1 "vmbr0" \
  --pve-network-vlan-1 "20" \
  test-vm
```

**Pros:**
- Full interactive shell
- Can run multiple commands
- Great for debugging
- Your SSH keys are mounted (read-only)

**Cons:**
- Slightly slower startup

### Method 3: Automated Test Script (`test-example.sh`)

Pre-configured test script with all the network configuration options:

```bash
# Edit the script with your credentials first
vim test-example.sh

# Or set environment variables
export PVE_URL="https://your-proxmox:8006"
export PVE_TOKEN_SECRET="your-secret"

# Run from within container
./test-docker.sh
./test-example.sh

# Or run directly
./test-quick.sh ./test-example.sh
```

## Testing Network Configuration

The new network configuration feature can be tested with:

```bash
docker-machine create \
  --driver pve \
  --pve-configure-network \
  --pve-network-bridge-0 "vmbr0" \
  --pve-network-vlan-0 "12" \
  --pve-network-bridge-1 "vmbr0" \
  --pve-network-vlan-1 "20" \
  <other-required-flags> \
  test-vm
```

### Verification

After creating a VM, verify the network configuration:

```bash
# SSH to your Proxmox host
ssh root@your-proxmox-host

# Check VM network configuration (replace <VM_ID> with actual ID)
qm config <VM_ID> | grep net

# Expected output:
# net0: virtio=XX:XX:XX:XX:XX:XX,bridge=vmbr0,tag=12
# net1: virtio=XX:XX:XX:XX:XX:XX,bridge=vmbr0,tag=20
```

## Environment Variables

All test scripts support these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PVE_URL` | Proxmox VE URL | `https://10.42.1.93:8006` |
| `PVE_TOKEN_ID` | API Token ID | `root@pam!rancher` |
| `PVE_TOKEN_SECRET` | API Token Secret | `YOUR_SECRET_HERE` |
| `PVE_RESOURCE_POOL` | Resource Pool | `k3s` |
| `PVE_TEMPLATE` | Template ID | `9000` |
| `PVE_ISO_DEVICE` | ISO Device | `scsi1` |
| `PVE_NETWORK_INTERFACE` | Network Interface | `net0` |
| `PVE_NETWORK_BRIDGE_0` | Bridge for NIC 0 | `vmbr0` |
| `PVE_NETWORK_VLAN_0` | VLAN for NIC 0 | `12` |
| `PVE_NETWORK_BRIDGE_1` | Bridge for NIC 1 | _(empty)_ |
| `PVE_NETWORK_VLAN_1` | VLAN for NIC 1 | _(empty)_ |

## Cleanup

### Remove Test VMs

```bash
# Inside container or via test-quick.sh
docker-machine rm test-vm

# Or remove all VMs
docker-machine rm -f $(docker-machine ls -q)
```

### Remove Docker Images

```bash
# Remove test image
docker rmi pve-driver-test

# Rebuild if needed
docker build -f Dockerfile.test -t pve-driver-test .
```

## Troubleshooting

### "bash: !packer: event not found" error

This happens when your token ID contains `!` which bash interprets as history expansion.

**Solution**: Use **single quotes** around the token ID:
```bash
--pve-token-id 'root@pam!packer'  # Good - single quotes
--pve-token-id "root@pam!packer"  # Bad - double quotes allow history expansion
```

Or disable history expansion temporarily:
```bash
set +H  # Disable history expansion
docker-machine create ... --pve-token-id "root@pam!packer" ...
set -H  # Re-enable history expansion
```

### "exec format error"

This means the binary was built for the wrong architecture (e.g., macOS binary in Linux container).

**Solution**: The test scripts now automatically build for Linux. If you built manually, rebuild:
```bash
# Build for Linux (required for Docker testing)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/docker-machine-driver-pve ./cmd/docker-machine-driver-pve
```

### Driver not found

Make sure you built it first:
```bash
# The test scripts build automatically, but if you need to rebuild:
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/docker-machine-driver-pve ./cmd/docker-machine-driver-pve
ls -la bin/docker-machine-driver-pve
```

### Permission denied on driver

The driver is mounted read-only. This is expected. If you need to modify it, rebuild:
```bash
# Exit container
exit
# Rebuild
task build:docker-machine-driver-pve
# Re-enter
./test-docker.sh
```

### SSH key issues

Your SSH keys are mounted read-only from `~/.ssh`. If you need different keys:
```bash
# Edit test-docker.sh and change the mount:
-v $HOME/.ssh:/root/.ssh:ro
# to
-v /path/to/your/keys:/root/.ssh:ro
```

### Connection to Proxmox fails

Make sure:
1. Your Proxmox URL is accessible from Docker
2. Your token has the correct permissions
3. TLS certificate is valid (or use `--pve-insecure-tls`)

## Advanced: CI/CD Integration

For CI/CD pipelines, use `test-quick.sh`:

```bash
#!/bin/bash
# .github/workflows/test.sh or similar

set -e

# Set credentials from CI secrets
export PVE_URL="$CI_PVE_URL"
export PVE_TOKEN_SECRET="$CI_PVE_TOKEN_SECRET"

# Run basic tests
./test-quick.sh docker-machine-driver-pve --version

# Run integration test if credentials available
if [ -n "$PVE_TOKEN_SECRET" ]; then
  ./test-quick.sh ./test-example.sh
fi
```

## Files Overview

- `Dockerfile.test` - Test container definition
- `test-docker.sh` - Interactive container launcher
- `test-quick.sh` - Quick command runner
- `test-example.sh` - Full integration test example
- `TESTING.md` - This file

## Development Workflow

1. Make changes to the Go code
2. Build: `task build:docker-machine-driver-pve`
3. Test interactively: `./test-docker.sh`
4. Run integration tests: `./test-quick.sh ./test-example.sh`
5. Commit when satisfied

No pollution of your local `/usr/local/bin` required!
