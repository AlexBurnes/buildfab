# Simple Dockerfile for buildfab container example
FROM alpine:latest

# Install buildfab binary (this would normally be built from source)
# For the example, we'll just create a simple script that acts like buildfab
RUN echo '#!/bin/sh' > /usr/local/bin/buildfab && \
    echo 'echo "buildfab version 1.0.0"' >> /usr/local/bin/buildfab && \
    echo 'if [ "$1" = "--version" ]; then echo "1.0.0"; exit 0; fi' >> /usr/local/bin/buildfab && \
    echo 'echo "buildfab: $@"' >> /usr/local/bin/buildfab && \
    chmod +x /usr/local/bin/buildfab

# Set working directory
WORKDIR /workspace

# Default command
CMD ["/usr/local/bin/buildfab", "--version"]
