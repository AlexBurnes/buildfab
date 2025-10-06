# Simple Dockerfile for buildfab container example
FROM alpine:latest

ARG VERSION

# Install buildfab binary (this would normally be built from source)
# For the example, we'll just create a simple script that acts like buildfab
RUN echo '#!/bin/sh' > /usr/local/bin/buildfab && \
    echo "VERSION=$VERSION" >> /usr/local/bin/buildfab && \
    echo 'echo "buildfab version $VERSION"' >> /usr/local/bin/buildfab && \
    echo 'if [ "$1" = "--version" ]; then echo "$VERSION"; exit 0; fi' >> /usr/local/bin/buildfab && \
    echo 'echo "buildfab: $@"' >> /usr/local/bin/buildfab && \
    chmod +x /usr/local/bin/buildfab

# Set working directory
WORKDIR /workspace

# Default command
CMD ["/usr/local/bin/buildfab", "--version"]
