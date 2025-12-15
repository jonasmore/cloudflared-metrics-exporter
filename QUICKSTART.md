# Quick Start Guide

Get up and running with cloudflared-metrics-exporter in 5 minutes.

## What is This?

Cloudflared Metrics Exporter is a tool for quick analysis of cloudflared metrics. It scrapes metrics from cloudflared's Prometheus endpoint and saves them in JSONL format for easy analysis and troubleshooting. The exported data can be imported into visualization tools like [cloudflared-metrics-exporter-vision](https://github.com/jonasmore/cloudflared-metrics-exporter-vision).

**Primary Use Cases:**
- Short-term monitoring and troubleshooting
- Quick metric analysis during incidents
- Long-term monitoring (with filtering, compression, and rotation)

**Best Deployment:**
- Run on the local machine or same local network as cloudflared for stability
- Alternative: Use WARP to tunnel to remote networks
- Alternative: Expose metrics endpoint with public hostname (**Cloudflare Access highly recommended for security**)

## Prerequisites

- Go 1.21+ (for building from source)
- A running cloudflared instance with metrics enabled
- Write access to a directory for storing metrics

## Step 1: Build the Exporter

```bash
cd cloudflared-metrics-exporter
make build
```

This creates the `cloudflared-metrics-exporter` binary in the current directory.

## Step 2: Verify Cloudflared Metrics

Ensure your cloudflared instance is exposing metrics (default port 2000):

```bash
curl http://localhost:2000/metrics
```

You should see Prometheus-format metrics output.

## Step 3: Run the Exporter

### Basic Usage (All Metrics)

```bash
./cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /tmp/metrics.jsonl
```

### Recommended Production Setup

```bash
./cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricsinterval 60s \
  --metricsfilter "cloudflared_tunnel_*,quic_client_*" \
  --metricscompress
```

## Step 4: Verify Output

Check that metrics are being written:

```bash
# View the file
tail -f /tmp/metrics.jsonl

# Pretty print with jq
tail -f /tmp/metrics.jsonl | jq .

# Count lines (each line = one metric sample)
wc -l /tmp/metrics.jsonl
```

## Step 5: Stop the Exporter

Press `Ctrl+C` to gracefully shutdown.

## Common Use Cases

### 1. Development/Testing

Export all metrics every 10 seconds:

```bash
./cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /tmp/dev-metrics.jsonl \
  --metricsinterval 10s
```

### 2. Production with Limited Disk

Export only critical metrics with compression:

```bash
./cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricsinterval 60s \
  --metricsfilter "cloudflared_tunnel_ha_connections,cloudflared_tunnel_total_requests,quic_client_smoothed_rtt" \
  --metricscompress
```

### 3. Monitoring Multiple Tunnels

Run separate instances for each tunnel:

```bash
# Tunnel 1
./cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/tunnel1-metrics.jsonl \
  --metricscompress &

# Tunnel 2
./cloudflared-metrics-exporter \
  --metrics localhost:2001 \
  --metricsfile /var/log/cloudflared/tunnel2-metrics.jsonl \
  --metricscompress &
```

### 4. Remote Monitoring

Monitor cloudflared from a different machine:

```bash
# Via local network
./cloudflared-metrics-exporter \
  --metrics 192.168.1.100:2000 \
  --metricsfile /var/log/cloudflared/remote-metrics.jsonl \
  --metricscompress

# Via WARP tunnel (if configured)
./cloudflared-metrics-exporter \
  --metrics tunnel-host.local:2000 \
  --metricsfile /var/log/cloudflared/remote-metrics.jsonl \
  --metricscompress

# Via public hostname with Cloudflare Access (preferred with authentication)
./cloudflared-metrics-exporter \
  --metrics https://metrics.example.com/metrics \
  --metricsfile /var/log/cloudflared/remote-metrics.jsonl \
  --cf-access-client-id <CLIENT_ID> \
  --cf-access-client-secret <CLIENT_SECRET> \
  --metricscompress
```

**Cloudflare Access Authentication (Optional, Highly Recommended)**: When exposing metrics endpoints publicly, it is **highly recommended** to protect them with Cloudflare Access. Use a [service token](https://developers.cloudflare.com/cloudflare-one/access-controls/service-credentials/service-tokens/#create-a-service-token) for authentication. The exporter will automatically add the required `CF-Access-Client-Id` and `CF-Access-Client-Secret` headers to requests.

**Note**: For remote monitoring, ensure stable network connectivity to avoid gaps in metrics collection.

## Analyzing Data

### View Latest Values

```bash
# Get latest HA connections count
jq -r 'select(.name == "cloudflared_tunnel_ha_connections") | .value' \
  /tmp/metrics.jsonl | tail -1

# Get all current QUIC RTT values
jq 'select(.name == "quic_client_smoothed_rtt")' \
  /tmp/metrics.jsonl | tail -4
```

### Calculate Averages

```bash
# Average RTT over time
jq -r 'select(.name == "quic_client_smoothed_rtt") | .value' \
  /tmp/metrics.jsonl | \
  awk '{sum+=$1; count++} END {print sum/count}'
```

### Find Changes

```bash
# See when HA connections changed
jq 'select(.name == "cloudflared_tunnel_ha_connections")' \
  /tmp/metrics.jsonl
```

## Visualizing Data

### Using cloudflared-metrics-exporter-vision

Import your metrics into the visualization tool:

```bash
# Clone the visualization tool
git clone https://github.com/jonasmore/cloudflared-metrics-exporter-vision
cd cloudflared-metrics-exporter-vision

# Follow the instructions in the vision repository
```

### Quick Analysis with jq and gnuplot

```bash
# Extract RTT values over time
jq -r 'select(.name == "quic_client_smoothed_rtt") | 
  "\(.timestamp) \(.value)"' metrics.jsonl > rtt.dat

# Plot with gnuplot
gnuplot -e "set terminal dumb; 
  set timefmt '%Y-%m-%dT%H:%M:%SZ'; 
  set xdata time; 
  plot 'rtt.dat' using 1:2 with lines"
```

## Running as a Service

### systemd (Linux)

1. Copy binary:
```bash
sudo cp cloudflared-metrics-exporter /usr/local/bin/
```

2. Create service file `/etc/systemd/system/cloudflared-metrics-exporter.service`:
```ini
[Unit]
Description=Cloudflared Metrics Exporter
After=network.target cloudflared.service

[Service]
Type=simple
User=cloudflared
ExecStart=/usr/local/bin/cloudflared-metrics-exporter \
  --metrics localhost:2000 \
  --metricsfile /var/log/cloudflared/metrics.jsonl \
  --metricsinterval 60s \
  --metricscompress
Restart=always

[Install]
WantedBy=multi-user.target
```

3. Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable cloudflared-metrics-exporter
sudo systemctl start cloudflared-metrics-exporter
```

### macOS (launchd)

Create `~/Library/LaunchAgents/com.cloudflare.metrics-exporter.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cloudflare.metrics-exporter</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/cloudflared-metrics-exporter</string>
        <string>--metrics</string>
        <string>localhost:2000</string>
        <string>--metricsfile</string>
        <string>/tmp/metrics.jsonl</string>
        <string>--metricscompress</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load:
```bash
launchctl load ~/Library/LaunchAgents/com.cloudflare.metrics-exporter.plist
```

## Troubleshooting

### "Connection refused" error

**Problem:** Cannot connect to metrics endpoint.

**Solution:**
```bash
# Check if cloudflared is running
ps aux | grep cloudflared

# Check if metrics port is open
netstat -an | grep 2000

# Try accessing metrics directly
curl http://localhost:2000/metrics
```

### "Permission denied" writing file

**Problem:** Cannot write to metrics file.

**Solution:**
```bash
# Create directory with correct permissions
sudo mkdir -p /var/log/cloudflared
sudo chown $USER:$USER /var/log/cloudflared

# Or use /tmp for testing
--metricsfile /tmp/metrics.jsonl
```

### File growing too large

**Problem:** Metrics file is consuming too much disk space.

**Solutions:**

1. Enable compression:
```bash
--metricscompress
```

2. Use filtering:
```bash
--metricsfilter "cloudflared_tunnel_*"
```

3. Rotate logs with logrotate:
```bash
# /etc/logrotate.d/cloudflared-metrics
/var/log/cloudflared/metrics.jsonl {
    daily
    rotate 7
    compress
    missingok
    notifempty
}
```

### Remote connection issues

**Problem:** Cannot connect to remote cloudflared instance.

**Solutions:**

1. **Local Network**: Ensure firewall allows access to port 2000
```bash
# Check if port is accessible
telnet 192.168.1.100 2000
curl http://192.168.1.100:2000/metrics
```

2. **WARP Tunnel**: Verify WARP is connected and routing is configured
```bash
# Check WARP status
warp-cli status
```

3. **Public Hostname**: Ensure Cloudflare Access is configured correctly
```bash
# Test with service token
curl -H "CF-Access-Client-Id: <CLIENT_ID>" \
     -H "CF-Access-Client-Secret: <CLIENT_SECRET>" \
     https://metrics.example.com/metrics
```

If you get a 403 error, verify:
- Service token is created in Cloudflare Access
- Service token is added to the Access policy for the metrics endpoint
- Client ID and Secret are correct

### Gaps in metrics data

**Problem:** Missing metrics during certain time periods.

**Causes:**
- Network connectivity issues (especially for remote monitoring)
- Exporter was stopped or crashed
- Disk full

**Solutions:**
- Use compression to reduce disk usage: `--metricscompress`
- Run exporter on same host as cloudflared for stability
- Set up systemd service with auto-restart

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check out [example queries](README.md#analyzing-exported-data)
- Set up log rotation for production use
- Integrate with your monitoring/analytics platform

## Getting Help

- Check logs: `--log-level debug`
- Review [troubleshooting section](README.md#troubleshooting)
- Open an issue on GitHub
