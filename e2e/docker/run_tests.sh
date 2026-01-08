#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "========================================"
echo "  Currier Docker Integration Tests"
echo "========================================"
echo ""
echo "Building and running tests in Docker..."
echo ""

cd "$PROJECT_ROOT"

# Build and run the Docker container
docker build -f e2e/docker/Dockerfile -t currier-capture-test .

echo ""
echo "Running capture integration tests..."
echo ""

# Run the tests
docker run --rm -it \
    --name currier-capture-test \
    --privileged \
    -e TERM=xterm-256color \
    currier-capture-test

echo ""
echo "Docker tests completed."
