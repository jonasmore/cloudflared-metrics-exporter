FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build
ARG VERSION=dev
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
    -o cloudflared-metrics-exporter

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/cloudflared-metrics-exporter /usr/local/bin/

# Create non-root user
RUN addgroup -g 1000 exporter && \
    adduser -D -u 1000 -G exporter exporter

USER exporter

ENTRYPOINT ["cloudflared-metrics-exporter"]
