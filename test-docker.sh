#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Docker-based Testing Environment ===${NC}\n"

# Step 1: Build the driver for Linux
echo -e "${YELLOW}Step 1: Building the driver for Linux...${NC}"
mkdir -p ./bin
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/docker-machine-driver-pve ./cmd/docker-machine-driver-pve
echo -e "${GREEN}✓ Driver built for Linux${NC}\n"

# Step 2: Build the test Docker image
echo -e "${YELLOW}Step 2: Building test Docker image...${NC}"
docker build -f Dockerfile.test -t pve-driver-test .
echo -e "${GREEN}✓ Test image built${NC}\n"

# Step 3: Run the container interactively
echo -e "${YELLOW}Step 3: Starting test container...${NC}"
echo -e "${GREEN}You are now in the isolated test environment!${NC}"
echo -e "${GREEN}The driver is available at: /usr/local/bin/docker-machine-driver-pve${NC}\n"
echo -e "Try running:"
echo -e "  ${YELLOW}docker-machine --version${NC}"
echo -e "  ${YELLOW}docker-machine-driver-pve --version${NC}"
echo -e "  ${YELLOW}docker-machine create --driver pve --help${NC}"
echo ""
echo -e "To create a test VM, use the example in /workspace/test-example.sh"
echo -e "or run: ${YELLOW}cat /workspace/test-example.sh${NC}"
echo ""

docker run -it --rm \
  -v $(pwd)/bin/docker-machine-driver-pve:/usr/local/bin/docker-machine-driver-pve:ro \
  -v $(pwd):/workspace \
  -v $HOME/.ssh:/root/.ssh:ro \
  pve-driver-test
