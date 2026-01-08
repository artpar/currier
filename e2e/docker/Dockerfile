FROM alpine:3.21

# Install dependencies
RUN apk add --no-cache \
    tmux \
    curl \
    bash \
    ca-certificates

# Set working directory
WORKDIR /app

# Copy pre-built binary (built locally for linux/arm64 or linux/amd64)
COPY currier-linux /usr/local/bin/currier
RUN chmod +x /usr/local/bin/currier

# Create test script
COPY e2e/docker/capture_test.sh /test/capture_test.sh
RUN chmod +x /test/capture_test.sh

ENTRYPOINT ["/test/capture_test.sh"]
