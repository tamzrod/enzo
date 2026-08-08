# Build stage for ENZO
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy source code
COPY . .

# Build ENZO
RUN CGO_ENABLED=0 GOOS=linux go build -o enzo ./cmd/enzo

# Build web UI
RUN CGO_ENABLED=0 GOOS=linux go build -o webui ./web

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/enzo ./enzo
COPY --from=builder /build/webui ./webui

# ENZO needs to be executable and findable
RUN chmod +x /app/enzo /app/webui && \
    ln -sf /app/enzo /usr/local/bin/enzo

EXPOSE 8080 8086 8088

CMD ["/app/webui"]
