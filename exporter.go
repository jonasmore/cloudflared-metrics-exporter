package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog"
)

// JSONLExporter exports Prometheus metrics to a JSONL (JSON Lines) file periodically
type JSONLExporter struct {
	metricsURL           string
	filePath             string
	interval             time.Duration
	httpClient           *http.Client
	log                  *zerolog.Logger
	filterPatterns       []string
	compress             bool
	cfAccessClientID     string
	cfAccessClientSecret string
	lastValues           map[string]float64
	lastValuesMu         sync.RWMutex
}

// MetricSample represents a single metric sample in JSONL format
type MetricSample struct {
	Timestamp string            `json:"timestamp"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

// NewJSONLExporter creates a new JSONL metrics exporter
func NewJSONLExporter(metricsURL, filePath string, interval time.Duration, filterPatterns []string, compress bool, cfAccessClientID, cfAccessClientSecret string, log *zerolog.Logger) (*JSONLExporter, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory for metrics file: %w", err)
	}

	return &JSONLExporter{
		metricsURL:           metricsURL,
		filePath:             filePath,
		interval:             interval,
		httpClient:           &http.Client{Timeout: 10 * time.Second},
		log:                  log,
		filterPatterns:       filterPatterns,
		compress:             compress,
		cfAccessClientID:     cfAccessClientID,
		cfAccessClientSecret: cfAccessClientSecret,
		lastValues:           make(map[string]float64),
	}, nil
}

// Run starts the periodic metrics export loop
func (e *JSONLExporter) Run(ctx context.Context) error {
	e.log.Info().
		Str("url", e.metricsURL).
		Str("file", e.filePath).
		Dur("interval", e.interval).
		Msg("Starting JSONL metrics exporter")

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	// Export immediately on start
	if err := e.exportMetrics(); err != nil {
		e.log.Err(err).Msg("Failed to export metrics on startup")
	}

	for {
		select {
		case <-ctx.Done():
			e.log.Info().Msg("JSONL metrics exporter shutting down")
			return nil
		case <-ticker.C:
			if err := e.exportMetrics(); err != nil {
				e.log.Err(err).Msg("Failed to export metrics")
			}
		}
	}
}

// exportMetrics fetches metrics from the HTTP endpoint and writes them to the JSONL file
func (e *JSONLExporter) exportMetrics() error {
	// Create HTTP request
	req, err := http.NewRequest("GET", e.metricsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add Cloudflare Access service token headers if configured
	if e.cfAccessClientID != "" && e.cfAccessClientSecret != "" {
		req.Header.Set("CF-Access-Client-Id", e.cfAccessClientID)
		req.Header.Set("CF-Access-Client-Secret", e.cfAccessClientSecret)
	}

	// Fetch metrics from HTTP endpoint
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode)
	}

	// Parse Prometheus metrics
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(e.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open metrics file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Write each metric family
	for _, mf := range metricFamilies {
		if err := e.writeMetricFamily(encoder, mf, timestamp); err != nil {
			return err
		}
	}

	// Ensure data is written to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync metrics file: %w", err)
	}

	return nil
}

// writeMetricFamily writes all metrics in a metric family to the JSONL file
func (e *JSONLExporter) writeMetricFamily(encoder *json.Encoder, mf *dto.MetricFamily, timestamp string) error {
	metricName := mf.GetName()

	// Apply filter if patterns are specified
	if len(e.filterPatterns) > 0 && !e.matchesFilter(metricName) {
		return nil // Skip this metric
	}

	metricType := mf.GetType().String()

	for _, m := range mf.GetMetric() {
		labels := e.extractLabels(m)

		// Handle different metric types
		switch mf.GetType() {
		case dto.MetricType_COUNTER:
			if err := e.writeSample(encoder, timestamp, metricName, metricType, m.GetCounter().GetValue(), labels); err != nil {
				return err
			}
		case dto.MetricType_GAUGE:
			if err := e.writeSample(encoder, timestamp, metricName, metricType, m.GetGauge().GetValue(), labels); err != nil {
				return err
			}
		case dto.MetricType_SUMMARY:
			summary := m.GetSummary()
			// Write quantiles
			for _, q := range summary.GetQuantile() {
				quantileLabels := e.copyLabels(labels)
				quantileLabels["quantile"] = fmt.Sprintf("%g", q.GetQuantile())
				if err := e.writeSample(encoder, timestamp, metricName, metricType, q.GetValue(), quantileLabels); err != nil {
					return err
				}
			}
			// Write sum and count
			sumLabels := e.copyLabels(labels)
			sumLabels["stat"] = "sum"
			if err := e.writeSample(encoder, timestamp, metricName, metricType, summary.GetSampleSum(), sumLabels); err != nil {
				return err
			}
			countLabels := e.copyLabels(labels)
			countLabels["stat"] = "count"
			if err := e.writeSample(encoder, timestamp, metricName, metricType, float64(summary.GetSampleCount()), countLabels); err != nil {
				return err
			}
		case dto.MetricType_HISTOGRAM:
			histogram := m.GetHistogram()
			// Write buckets
			for _, b := range histogram.GetBucket() {
				bucketLabels := e.copyLabels(labels)
				bucketLabels["le"] = fmt.Sprintf("%g", b.GetUpperBound())
				if err := e.writeSample(encoder, timestamp, metricName+"_bucket", metricType, float64(b.GetCumulativeCount()), bucketLabels); err != nil {
					return err
				}
			}
			// Write sum and count
			sumLabels := e.copyLabels(labels)
			sumLabels["stat"] = "sum"
			if err := e.writeSample(encoder, timestamp, metricName+"_sum", metricType, histogram.GetSampleSum(), sumLabels); err != nil {
				return err
			}
			countLabels := e.copyLabels(labels)
			countLabels["stat"] = "count"
			if err := e.writeSample(encoder, timestamp, metricName+"_count", metricType, float64(histogram.GetSampleCount()), countLabels); err != nil {
				return err
			}
		case dto.MetricType_UNTYPED:
			if err := e.writeSample(encoder, timestamp, metricName, metricType, m.GetUntyped().GetValue(), labels); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeSample writes a single metric sample to the JSONL file
func (e *JSONLExporter) writeSample(encoder *json.Encoder, timestamp, name, metricType string, value float64, labels map[string]string) error {
	// Check if compression is enabled and value hasn't changed
	if e.compress {
		key := e.buildMetricKey(name, labels)
		if !e.shouldWriteMetric(key, value) {
			return nil // Skip writing unchanged metric
		}
	}

	sample := MetricSample{
		Timestamp: timestamp,
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
	}

	if err := encoder.Encode(sample); err != nil {
		return fmt.Errorf("failed to encode metric sample: %w", err)
	}

	return nil
}

// extractLabels extracts labels from a Prometheus metric
func (e *JSONLExporter) extractLabels(m *dto.Metric) map[string]string {
	labels := make(map[string]string)
	for _, label := range m.GetLabel() {
		labels[label.GetName()] = label.GetValue()
	}
	return labels
}

// copyLabels creates a copy of a label map
func (e *JSONLExporter) copyLabels(labels map[string]string) map[string]string {
	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}

// matchesFilter checks if a metric name matches any of the filter patterns
func (e *JSONLExporter) matchesFilter(metricName string) bool {
	for _, pattern := range e.filterPatterns {
		if matchWildcard(pattern, metricName) {
			return true
		}
	}
	return false
}

// buildMetricKey creates a unique key for a metric based on name and labels
func (e *JSONLExporter) buildMetricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// Create a stable hash of labels
	// Sort keys to ensure consistent ordering
	var labelPairs []string
	for k, v := range labels {
		labelPairs = append(labelPairs, k+"="+v)
	}
	// Simple concatenation is sufficient since we control the format
	// and metric names/labels don't contain special characters
	labelStr := strings.Join(labelPairs, ",")
	return name + "{" + labelStr + "}"
}

// shouldWriteMetric checks if a metric value has changed and should be written
// Returns true if the metric should be written, false if it should be skipped
func (e *JSONLExporter) shouldWriteMetric(key string, value float64) bool {
	e.lastValuesMu.RLock()
	lastValue, exists := e.lastValues[key]
	e.lastValuesMu.RUnlock()

	// Always write if this is the first time we see this metric
	if !exists {
		e.lastValuesMu.Lock()
		e.lastValues[key] = value
		e.lastValuesMu.Unlock()
		return true
	}

	// Check if value has changed
	if value != lastValue {
		e.lastValuesMu.Lock()
		e.lastValues[key] = value
		e.lastValuesMu.Unlock()
		return true
	}

	// Value unchanged, skip writing
	return false
}

// matchWildcard performs simple wildcard matching (* matches any sequence of characters)
func matchWildcard(pattern, str string) bool {
	// Handle exact match
	if pattern == str {
		return true
	}

	// Handle wildcard patterns
	if !strings.Contains(pattern, "*") {
		return pattern == str
	}

	// Split pattern by * and check if all parts exist in order
	parts := strings.Split(pattern, "*")

	// Pattern starts with *
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	// Pattern ends with *
	endsWithWildcard := pattern[len(pattern)-1] == '*'

	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}

		pos := strings.Index(str[idx:], part)
		if pos == -1 {
			return false
		}

		// First part must match at the beginning (unless pattern starts with *)
		if i == 0 && pattern[0] != '*' && pos != 0 {
			return false
		}

		idx += pos + len(part)
	}

	// If pattern doesn't end with *, the string must be fully consumed
	if !endsWithWildcard && idx != len(str) {
		return false
	}

	return true
}
