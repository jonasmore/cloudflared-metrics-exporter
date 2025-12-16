# Cloudflared Metrics Exporter

[![Docker Pulls](https://img.shields.io/docker/pulls/jonasmore/cloudflared-metrics-exporter?logo=docker&logoColor=white)](https://hub.docker.com/r/jonasmore/cloudflared-metrics-exporter)
[![Docker Image Size](https://img.shields.io/docker/image-size/jonasmore/cloudflared-metrics-exporter/latest?logo=docker&logoColor=white)](https://hub.docker.com/r/jonasmore/cloudflared-metrics-exporter)
[![GitHub Release](https://img.shields.io/github/v/release/jonasmore/cloudflared-metrics-exporter?logo=github)](https://github.com/jonasmore/cloudflared-metrics-exporter/releases)
[![GitHub Downloads](https://img.shields.io/github/downloads/jonasmore/cloudflared-metrics-exporter/total?logo=github)](https://github.com/jonasmore/cloudflared-metrics-exporter/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jonasmore/cloudflared-metrics-exporter?logo=go)](go.mod)
[![Platform Support](https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20windows-lightgrey)](https://github.com/jonasmore/cloudflared-metrics-exporter/releases)

A standalone Go application that exports Cloudflare Tunnel (cloudflared) Prometheus metrics to JSONL (JSON Lines) format for quick analysis and monitoring. This tool scrapes metrics from cloudflared's `/metrics` endpoint and saves them in JSONL format for easy analysis, visualization, and troubleshooting.

## Overview

This project is designed for quick analysis of cloudflared metrics. It scrapes metrics and saves them in JSONL format, which can then be visualized using [cloudflared-metrics-exporter-vision](https://cloudflared-metrics-exporter-vision.jonasmore.dev/). While primarily intended for monitoring and troubleshooting over short periods, it can also be used for long-term monitoring with filtering, compression, and rotation/retention settings.

## Features

- 📊 **Export Prometheus metrics to JSONL** - Easy-to-parse time-series data
- 🔍 **Metric filtering** - Export only the metrics you need with wildcard support
- 🗜️ **Compression mode** - Save 70-90% disk space by only writing changed values
- ⚡ **Lightweight** - Minimal resource usage, runs alongside cloudflared
- 🔄 **Configurable intervals** - Control export frequency (default: 60s)
- 🛡️ **Graceful shutdown** - Handles SIGINT/SIGTERM properly

## Deployment Recommendations

For best results, run the exporter on the local machine or in the same local network as the cloudflared instance to ensure stable connectivity to the metrics endpoint. Alternative deployment options include:

- **WARP**: Use WARP to tunnel to remote networks
- **Public Hostname**: Expose the metrics endpoint with a public hostname (**Cloudflare Access with service tokens highly recommended for security**)

## Installation

### Docker

Pull the latest image from GitHub Container Registry or Docker Hub:

```bash
# From GitHub Container Registry
docker pull ghcr.io/jonasmore/cloudflared-metrics-exporter:latest

# From Docker Hub
docker pull jonasmore/cloudflared-metrics-exporter:latest
```

Run the container:

```bash
docker run -d \
  --name metrics-exporter \
  -v /var/log/cloudflared:/var/log/cloudflared \
  ghcr.io/jonasmore/cloudflared-metrics-exporter:latest \
  --metrics host.docker.internal:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricscompress
```

### Binary Release

Download pre-built binaries for your platform from the [releases page](https://github.com/jonasmore/cloudflared-metrics-exporter/releases).

Available platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

Each release includes SHA256 checksums for verification.

### From Source

```bash
git clone https://github.com/jonasmore/cloudflared-metrics-exporter
cd cloudflared-metrics-exporter
go build -o cloudflared-metrics-exporter
```

## How It Works

The exporter operates independently from cloudflared, fetching metrics via HTTP:

```
┌─────────────────┐
│   cloudflared   │
│  (Port 2000)    │
│  /metrics       │
└────────┬────────┘
         │ HTTP GET
         │ (Prometheus format)
         ▼
┌─────────────────────────┐
│ cloudflared-metrics-    │
│      exporter           │
│                         │
│  • Fetch metrics        │
│  • Parse Prometheus     │
│  • Apply filters        │
│  • Check compression    │
│  • Write JSONL          │
└────────┬────────────────┘
         │
         ▼
┌─────────────────┐
│  metrics.jsonl  │
│                 │
│  {"timestamp":  │
│   "2025-...",   │
│   "name":"...", │
│   "value":...}  │
└─────────────────┘
```

### Key Characteristics

- **HTTP-Based Collection**: Fetches metrics from cloudflared's `/metrics` endpoint
- **No Configuration Changes**: Works with any cloudflared instance exposing Prometheus metrics
- **Standalone Deployment**: No cloudflared code dependencies, can run on same or different host
- **Minimal Footprint**: ~10-20 MB RAM usage

## Usage

### Basic Usage

Export all metrics from a cloudflared instance:

```bash
cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl
```

### With Compression (Recommended)

Save disk space by only writing changed values:

```bash
cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricscompress
```

### With Filtering

Export only specific metrics:

```bash
cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricsfilter "cloudflared_tunnel_*,quic_client_*" \
  --metricscompress
```

### Custom Interval

Export every 30 seconds:

```bash
cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricsinterval 30s
```

### With Cloudflare Access Authentication (Optional, Highly Recommended)

For metrics endpoints protected by Cloudflare Access, use a service token. This is **highly recommended** when exposing metrics endpoints publicly for security:

```bash
cloudflared-metrics-exporter \
  --metrics https://metrics.example.com \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --cf-access-client-id <CLIENT_ID> \
  --cf-access-client-secret <CLIENT_SECRET> \
  --metricscompress
```

**Learn more**: [Create a Cloudflare Access service token](https://developers.cloudflare.com/cloudflare-one/access-controls/service-credentials/service-tokens/#create-a-service-token)

## Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--metrics` | Metrics endpoint address (e.g., `localhost:2000` or `http://localhost:2000`) | - | Yes |
| `--metricsfile` | Path to the JSONL output file | - | Yes |
| `--metricsinterval` | Export frequency | `60s` | No |
| `--metricsfilter` | Comma-separated metric patterns (supports `*` wildcards) | All metrics | No |
| `--metricscompress` | Enable change-only export mode | `false` | No |
| `--cf-access-client-id` | Cloudflare Access service token Client ID | - | No |
| `--cf-access-client-secret` | Cloudflare Access service token Client Secret | - | No |
| `--log-level` | Log level (`debug`, `info`, `warn`, `error`) | `info` | No |

## Environment Variables

All flags can be set via environment variables:

```bash
export METRICS_ENDPOINT=localhost:2000
export METRICS_FILE=/var/log/cloudflared/metrics.jsonl
export METRICS_INTERVAL=30s
export METRICS_FILTER="cloudflared_tunnel_*,quic_client_*"
export METRICS_COMPRESS=true
export CF_ACCESS_CLIENT_ID=<CLIENT_ID>
export CF_ACCESS_CLIENT_SECRET=<CLIENT_SECRET>
export LOG_LEVEL=info

cloudflared-metrics-exporter
```

## Output Format

Each line in the output file is a valid JSON object:

```json
{"timestamp":"2025-12-14T22:30:15Z","name":"cloudflared_tunnel_ha_connections","type":"GAUGE","value":4,"labels":{}}
{"timestamp":"2025-12-14T22:30:15Z","name":"quic_client_smoothed_rtt","type":"GAUGE","value":7,"labels":{"conn_index":"0"}}
{"timestamp":"2025-12-14T22:30:15Z","name":"cloudflared_tunnel_response_by_code","type":"COUNTER","value":4533,"labels":{"status_code":"200"}}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string | ISO 8601 timestamp (UTC) when the metric was collected |
| `name` | string | Metric name |
| `type` | string | Metric type: `COUNTER`, `GAUGE`, `HISTOGRAM`, `SUMMARY`, or `UNTYPED` |
| `value` | number | Metric value |
| `labels` | object | Key-value pairs of metric labels |

## Compression Mode

When `--metricscompress` is enabled, metrics are only written when their value changes:

**Without compression:**
```json
{"timestamp":"2025-12-14T22:30:00Z","name":"cloudflared_tunnel_ha_connections","value":4}
{"timestamp":"2025-12-14T22:30:30Z","name":"cloudflared_tunnel_ha_connections","value":4}
{"timestamp":"2025-12-14T22:31:00Z","name":"cloudflared_tunnel_ha_connections","value":4}
{"timestamp":"2025-12-14T22:31:30Z","name":"cloudflared_tunnel_ha_connections","value":3}
```

**With compression:**
```json
{"timestamp":"2025-12-14T22:30:00Z","name":"cloudflared_tunnel_ha_connections","value":4}
{"timestamp":"2025-12-14T22:31:30Z","name":"cloudflared_tunnel_ha_connections","value":3}
```

**Storage savings:** Typically 70-90% smaller files for production workloads.

## Metric Filtering

Use wildcards to export only specific metrics:

```bash
# Export only tunnel metrics
--metricsfilter "cloudflared_tunnel_*"

# Export tunnel and QUIC metrics
--metricsfilter "cloudflared_tunnel_*,quic_client_*"

# Export all Go memory stats
--metricsfilter "go_memstats_*"

# Export specific metrics
--metricsfilter "cloudflared_tunnel_ha_connections,quic_client_smoothed_rtt"
```

### Common Filter Patterns

| Pattern | Matches |
|---------|---------|
| `cloudflared_tunnel_*` | All tunnel-specific metrics |
| `quic_client_*` | All QUIC client metrics |
| `go_memstats_*` | Go runtime memory statistics |
| `process_*` | Process-level metrics |
| `*_total` | All counter metrics ending with "_total" |

## Deployment Scenarios

### 1. Side-by-Side (Same Host)

Run the exporter on the same machine as cloudflared for maximum stability:

```
┌──────────────────────────────┐
│         Server               │
│                              │
│  ┌────────────┐              │
│  │cloudflared │              │
│  │:2000       │              │
│  └─────┬──────┘              │
│        │                     │
│        ▼                     │
│  ┌────────────────────┐     │
│  │metrics-exporter    │     │
│  └────────┬───────────┘     │
│           │                  │
│           ▼                  │
│  /var/log/cloudflared/       │
│    metrics.jsonl             │
└──────────────────────────────┘
```

### 2. Remote Monitoring

Monitor multiple cloudflared instances from a central host:

```
┌──────────────┐      ┌──────────────────┐
│ Server A     │      │ Monitoring Host  │
│              │      │                  │
│ cloudflared  │◄─────┤ metrics-exporter │
│ :2000        │ HTTP │                  │
└──────────────┘      └────────┬─────────┘
                               │
┌──────────────┐               │
│ Server B     │               │
│              │               │
│ cloudflared  │◄──────────────┘
│ :2000        │ HTTP
└──────────────┘
```

**Note**: For remote monitoring, ensure network connectivity via:
- Local network access
- WARP tunnel
- Public hostname with Cloudflare Access (**highly recommended** - use service tokens for authentication)

**Example with Cloudflare Access (Recommended for Public Endpoints)**:
```bash
./cloudflared-metrics-exporter \
  --metrics https://metrics.example.com \
  --metricsfile /var/log/cloudflared/remote-metrics.jsonl \
  --cf-access-client-id <CLIENT_ID> \
  --cf-access-client-secret <CLIENT_SECRET> \
  --metricscompress
```

See [Cloudflare Access service tokens documentation](https://developers.cloudflare.com/cloudflare-one/access-controls/service-credentials/service-tokens/#create-a-service-token) for setup instructions.

### 3. Container Deployment

Deploy alongside cloudflared in Docker or Kubernetes:

```yaml
version: '3'
services:
  cloudflared:
    image: cloudflare/cloudflared
    command: tunnel run
    ports:
      - "2000:2000"
  
  metrics-exporter:
    image: ghcr.io/jonasmore/cloudflared-metrics-exporter:latest
    command:
      - --metrics=cloudflared:2000
      - --metricsfile=/data/metrics.jsonl
      - --metricscompress
    volumes:
      - ./metrics:/data
    depends_on:
      - cloudflared
```

## Running as a Service

### systemd (Linux)

Create `/etc/systemd/system/cloudflared-metrics-exporter.service`:

```ini
[Unit]
Description=Cloudflared Metrics Exporter
After=network.target cloudflared.service
Wants=cloudflared.service

[Service]
Type=simple
User=cloudflared
Group=cloudflared
ExecStart=/usr/local/bin/cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricsinterval 60s \
  --metricsfilter "cloudflared_tunnel_*,quic_client_*" \
  --metricscompress
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable cloudflared-metrics-exporter
sudo systemctl start cloudflared-metrics-exporter
sudo systemctl status cloudflared-metrics-exporter
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o cloudflared-metrics-exporter

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/cloudflared-metrics-exporter /usr/local/bin/
ENTRYPOINT ["cloudflared-metrics-exporter"]
```

Run:

```bash
docker build -t cloudflared-metrics-exporter .
docker run -d \
  --name metrics-exporter \
  -v /var/log/cloudflared:/var/log/cloudflared \
  cloudflared-metrics-exporter \
  --metrics host.docker.internal:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricscompress
```

## Analyzing Exported Data

### View Real-Time Metrics

```bash
tail -f /var/log/cloudflared/metrics.jsonl | jq .
```

### Query Specific Metrics

```bash
# Find all HA connection changes
jq 'select(.name == "cloudflared_tunnel_ha_connections")' metrics.jsonl

# Get latest value of a metric
jq 'select(.name == "cloudflared_tunnel_ha_connections") | .value' metrics.jsonl | tail -1

# Filter by time range
jq 'select(.timestamp >= "2025-12-14T20:00:00Z" and .timestamp <= "2025-12-14T21:00:00Z")' metrics.jsonl
```

### Calculate Statistics

```bash
# Average RTT over time
jq -r 'select(.name == "quic_client_smoothed_rtt") | .value' metrics.jsonl | \
  awk '{sum+=$1; count++} END {print sum/count}'

# Count HTTP status codes
jq -r 'select(.name == "cloudflared_tunnel_response_by_code") | 
  "\(.labels.status_code): \(.value)"' metrics.jsonl | sort | uniq -c
```

### Import to Database

**PostgreSQL:**

```sql
CREATE TABLE metrics (
    timestamp TIMESTAMPTZ,
    name TEXT,
    type TEXT,
    value DOUBLE PRECISION,
    labels JSONB
);

COPY metrics FROM PROGRAM 
'cat /var/log/cloudflared/metrics.jsonl' 
WITH (FORMAT csv, DELIMITER E'\x01', QUOTE E'\x02');
```

**ClickHouse:**

```sql
CREATE TABLE metrics (
    timestamp DateTime,
    name String,
    type String,
    value Float64,
    labels Map(String, String)
) ENGINE = MergeTree()
ORDER BY (name, timestamp);

-- Import from JSONL
cat metrics.jsonl | clickhouse-client --query="INSERT INTO metrics FORMAT JSONEachRow"
```

## Log Rotation and Retention

For long-term monitoring, implement log rotation to manage disk space:

### Using logrotate (Linux)

Create `/etc/logrotate.d/cloudflared-metrics`:

```
/var/log/cloudflared/metrics.jsonl {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 cloudflared cloudflared
    postrotate
        systemctl reload cloudflared-metrics-exporter 2>/dev/null || true
    endscript
}
```

### Manual Rotation Script

```bash
#!/bin/bash
# rotate-metrics.sh
METRICS_FILE="/var/log/cloudflared/metrics.jsonl"
ARCHIVE_DIR="/var/log/cloudflared/archive"
RETENTION_DAYS=30

# Create archive directory
mkdir -p "$ARCHIVE_DIR"

# Rotate if file exists and is not empty
if [ -s "$METRICS_FILE" ]; then
    TIMESTAMP=$(date +%Y%m%d-%H%M%S)
    mv "$METRICS_FILE" "$ARCHIVE_DIR/metrics-$TIMESTAMP.jsonl"
    gzip "$ARCHIVE_DIR/metrics-$TIMESTAMP.jsonl"
fi

# Delete old archives
find "$ARCHIVE_DIR" -name "metrics-*.jsonl.gz" -mtime +$RETENTION_DAYS -delete
```

Add to crontab for daily rotation:
```bash
0 0 * * * /usr/local/bin/rotate-metrics.sh
```

## Troubleshooting

### Cannot connect to metrics endpoint

```bash
# Check if cloudflared metrics server is running
curl http://localhost:2000/metrics

# Check cloudflared configuration
cloudflared tunnel info <tunnel-name>
```

### Permission denied writing to file

```bash
# Ensure directory exists and has correct permissions
sudo mkdir -p /var/log/cloudflared
sudo chown cloudflared:cloudflared /var/log/cloudflared
sudo chmod 755 /var/log/cloudflared
```

### High disk usage

Enable compression mode:

```bash
--metricscompress
```

Or use more aggressive filtering:

```bash
--metricsfilter "cloudflared_tunnel_ha_connections,cloudflared_tunnel_total_requests"
```

### Metrics not updating

Check the exporter logs:

```bash
# If running as systemd service
sudo journalctl -u cloudflared-metrics-exporter -f

# If running in foreground
cloudflared-metrics-exporter --log-level debug ...
```

## Performance

### Resource Usage

- **CPU:** < 1% on average
- **Memory:** ~10-20 MB
- **Disk I/O:** Depends on interval and number of metrics
  - Without compression: ~2-6 MB/hour (100 metrics @ 60s interval)
  - With compression: ~0.1-1 MB/hour (70-90% reduction)

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

Apache 2.0 - See LICENSE file for details.

## Related Projects

- [cloudflared](https://github.com/cloudflare/cloudflared) - Cloudflare Tunnel client
- [Prometheus](https://prometheus.io/) - Monitoring system and time series database

## Support

For issues or questions:
1. Check the [troubleshooting section](#troubleshooting)
2. Review [cloudflared metrics documentation](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/monitor-tunnels/metrics/)
3. Open an issue on GitHub
