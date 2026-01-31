#!/bin/bash
# Quick test script - runs a single command in Docker without entering the container
set -e

# Build the driver for Linux first
echo "Building driver for Linux..."
mkdir -p ./bin
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/docker-machine-driver-pve ./cmd/docker-machine-driver-pve

# Build test image if it doesn't exist
if ! docker images pve-driver-test | grep -q pve-driver-test; then
  echo "Building test Docker image..."
  docker build -f Dockerfile.test -t pve-driver-test .
fi

# Run the command passed as arguments, or show help if no args
if [ $# -eq 0 ]; then
  echo ""
  echo "Usage: $0 <command>"
  echo ""
  echo "Examples:"
  echo "  $0 docker-machine --version"
  echo "  $0 docker-machine-driver-pve --version"
  echo "  $0 docker-machine create --driver pve --help"
  echo ""
  echo "To run the full test example:"
  echo "  $0 ./test-example.sh"
  echo ""
  echo "To enter the container interactively:"
  echo "  ./test-docker.sh"
  echo ""
  exit 0
fi

# Run the command in Docker
docker run --rm \
  -v $(pwd)/bin/docker-machine-driver-pve:/usr/local/bin/docker-machine-driver-pve:ro \
  -v $(pwd):/workspace \
  -v $HOME/.ssh:/root/.ssh:ro \
  -e PVE_URL \
  -e PVE_TOKEN_ID \
  -e PVE_TOKEN_SECRET \
  -e PVE_RESOURCE_POOL \
  -e PVE_TEMPLATE \
  -e PVE_ISO_DEVICE \
  -e PVE_NETWORK_INTERFACE \
  -e PVE_NETWORK_BRIDGE_0 \
  -e PVE_NETWORK_VLAN_0 \
  -e PVE_NETWORK_BRIDGE_1 \
  -e PVE_NETWORK_VLAN_1 \
  pve-driver-test \
  -c "$*"
