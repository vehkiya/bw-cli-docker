# Stage 1: "downloader"
# This stage downloads and extracts the Bitwarden CLI binary.
FROM debian:sid AS downloader
ARG BW_CLI_VERSION=2025.6.1
ENV BW_CLI_VERSION=${BW_CLI_VERSION}
RUN apt-get update && \
    apt-get install -y curl unzip && \
    curl -fL "https://github.com/bitwarden/clients/releases/download/cli-v${BW_CLI_VERSION}/bw-linux-${BW_CLI_VERSION}.zip" -o bw.zip && \
    unzip bw.zip && \
    chmod +x bw

# --------------------------------------------------------------------

# Stage 2: "builder"
# This stage compiles our Go entrypoint program into a static binary.
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go mod init entrypoint
RUN go mod tidy
# Build a static, CGO-disabled binary to ensure it runs on any minimal base image.
RUN CGO_ENABLED=0 go build -o /entrypoint .

# --------------------------------------------------------------------

# Stage 3: "final"
# This is our final, minimal, shell-less image.
# Using 'static' ensures there are no shells or other binaries.
FROM gcr.io/distroless/cc-debian12

# Copy the Bitwarden CLI from the 'downloader' stage.
COPY --from=downloader /bw /usr/local/bin/bw

# Copy the compiled Go entrypoint from the 'builder' stage.
COPY --from=builder /entrypoint /entrypoint

USER nonroot:nonroot

# Set the entrypoint to our compiled Go program.98
EXPOSE 8080
ENTRYPOINT ["/entrypoint"]