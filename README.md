# Upfluence Stats

A lightweight, flexible metrics library for Go that provides a unified interface for collecting application metrics with support for multiple backends including Prometheus and expvar.

## Features

- **Multiple metric types**: Counters, Gauges, Histograms, and Timers
- **Label/tag support**: Multi-dimensional metrics with labels
- **Scoped metrics**: Hierarchical metric organization with namespace and tag inheritance
- **Multiple backends**: Prometheus, expvar, or custom collectors
- **Thread-safe**: Concurrent metric updates without locks
- **Zero allocations**: Optimized for high-performance applications
- **Flexible histograms**: Configurable bucket boundaries

## Installation

```bash
go get github.com/upfluence/stats
```

## Quick Start

```go
package main

import (
    "github.com/upfluence/stats"
    "github.com/upfluence/stats/prometheus"
)

func main() {
    // Create a Prometheus collector
    collector := prometheus.NewCollector()

    // Create a root scope
    scope := stats.RootScope(collector)

    // Create and use metrics
    counter := scope.Counter("requests_total")
    counter.Inc()

    gauge := scope.Gauge("active_connections")
    gauge.Update(42)

    histogram := scope.Histogram("request_duration_seconds")
    histogram.Record(0.123)
}
```

## Metric Types

### Counter

Counters are monotonically increasing values, ideal for tracking totals.

```go
// Simple counter
counter := scope.Counter("http_requests_total")
counter.Inc()
counter.Add(5)

// Counter with labels
counterVec := scope.CounterVector("http_requests_total", []string{"method", "status"})
counterVec.WithLabels("GET", "200").Inc()
counterVec.WithLabels("POST", "201").Add(10)
```

### Gauge

Gauges represent values that can go up or down.

```go
// Simple gauge
gauge := scope.Gauge("memory_usage_bytes")
gauge.Update(1024000)

// Gauge with labels
gaugeVec := scope.GaugeVector("cpu_usage_percent", []string{"core"})
gaugeVec.WithLabels("0").Update(75)
gaugeVec.WithLabels("1").Update(82)
```

### Histogram

Histograms track distributions of values across buckets.

```go
// Default histogram (uses standard buckets)
histogram := scope.Histogram("response_time_seconds")
histogram.Record(0.234)

// Custom buckets
histogram := scope.Histogram("response_time_seconds",
    stats.StaticBuckets([]float64{0.01, 0.05, 0.1, 0.5, 1.0}))
histogram.Record(0.123)

// Histogram with labels
histVec := scope.HistogramVector("request_duration_seconds",
    []string{"endpoint", "method"})
histVec.WithLabels("/api/users", "GET").Record(0.045)
```

### Timer

Timers are convenience wrappers around histograms for measuring durations.

```go
timer := stats.NewTimer(scope, "operation_duration")
stopwatch := timer.Start()
// ... perform operation ...
stopwatch.Stop() // Records duration in seconds

// Timer with custom suffix and buckets
timer := stats.NewTimer(scope, "db_query",
    stats.WithTimerSuffix("_ms"),
    stats.WithHistogramOptions(
        stats.StaticBuckets([]float64{1, 5, 10, 50, 100})))
```

### Instrument

Instruments provide automatic instrumentation for function execution, combining multiple metrics into a single convenient interface. They automatically track:

- **Started counter**: Number of times the operation started (optional)
- **Finished counter**: Number of completions, labeled by status (success/failed)
- **Duration histogram**: Execution time distribution

```go
// Basic instrument
instrument := stats.NewInstrument(scope, "api_request")

err := instrument.Exec(func() error {
    // Your operation here
    return processRequest()
})
// Automatically records:
// - api_request_started_total (counter)
// - api_request_total{status="success"} or {status="failed"} (counter)
// - api_request_duration_seconds (histogram)

// Instrument with labels
instrumentVec := stats.NewInstrumentVector(
    scope,
    "database_query",
    []string{"table", "operation"},
)

err := instrumentVec.WithLabels("users", "select").Exec(func() error {
    return db.Query("SELECT * FROM users")
})

// Custom options
instrument := stats.NewInstrument(scope, "task",
    // Disable the started counter
    stats.DisableStartedCounter(),

    // Custom status label name (default: "status")
    stats.WithCounterLabel("result"),

    // Custom error formatter
    stats.WithFormatter(func(err error) string {
        if err == nil {
            return "ok"
        }
        return "error"
    }),

    // Custom timer options
    stats.WithTimerOptions(
        stats.WithTimerSuffix("_duration_ms"),
        stats.WithHistogramOptions(
            stats.StaticBuckets([]float64{10, 50, 100, 500, 1000}),
        ),
    ),
)
```

**Error Formatters**: Customize how errors are categorized in the status label:

```go
// Advanced error formatter using error types
formatter := func(err error) string {
    if err == nil {
        return "success"
    }

    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return "timeout"
    case errors.Is(err, context.Canceled):
        return "canceled"
    case isValidationError(err):
        return "validation_error"
    default:
        return "internal_error"
    }
}

instrument := stats.NewInstrument(
    scope,
    "operation",
    stats.WithFormatter(formatter),
)
```

**Generated Metrics**:

For an instrument named `"api_request"`, the following metrics are created:

- `api_request_started_total` - Counter tracking how many times the operation started (useful for computing in-flight count: `started_total - sum(total)`)
- `api_request_total{status="..."}` - Counter tracking completions by status
- `api_request_duration_seconds` - Histogram of execution durations

## Scopes

Scopes allow you to organize metrics hierarchically with namespace prefixes and tag inheritance.

```go
// Create a child scope with namespace and tags
dbScope := scope.Scope("database", map[string]string{
    "service": "user-api",
    "env": "production",
})

// Metrics created from this scope inherit the namespace and tags
counter := dbScope.Counter("queries")
// Results in metric name: database_queries
// with tags: service=user-api, env=production

// Nested scopes
redisScope := dbScope.Scope("redis", map[string]string{
    "cluster": "cache-01",
})
// Metrics will have namespace: database_redis
// and tags: service=user-api, env=production, cluster=cache-01
```

## Collectors

### Prometheus Collector

Export metrics to Prometheus:

```go
import (
    "github.com/upfluence/stats/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

collector := prometheus.NewCollector()
scope := stats.RootScope(collector)

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":8080", nil)
```

### Prometheus Push Gateway

Push metrics to a Prometheus Push Gateway:

```go
import (
    "github.com/upfluence/stats/prometheus"
    "github.com/upfluence/log"
    "time"
)

exporter := prometheus.NewExporter(
    log.Default,
    "http://pushgateway:9091",
    "my-application",
    prometheus.WithInterval(10 * time.Second),
    prometheus.WithGrouping("instance", "server-01"),
)
defer exporter.Close()
```

### Expvar Collector

Export metrics via Go's expvar package:

```go
import "github.com/upfluence/stats/expvar"

collector := expvar.NewCollector()
scope := stats.RootScope(collector)

// Metrics are automatically available at /debug/vars
```

### Multiple Collectors

Use multiple collectors simultaneously:

```go
import "github.com/upfluence/stats/multi"

promCollector := prometheus.NewCollector()
expvarCollector := expvar.NewCollector()

collector := multi.WrapCollectors(promCollector, expvarCollector)
scope := stats.RootScope(collector)

// Metrics are sent to both collectors
```

## Advanced Features

### NoopScope

For testing or conditional metric collection:

```go
var scope stats.Scope

if metricsEnabled {
    scope = stats.RootScope(collector)
} else {
    scope = stats.NoopScope
}

// Code works the same, but NoopScope doesn't collect anything
scope.Counter("requests").Inc()
```

### Multi-Incarnation Scope

Track metrics across service restarts while maintaining historical data:

```go
scope := stats.NewMultiIncarnationScope(
    collector,
    "app_version",
    "1.2.3",
)

// Metrics include the incarnation label automatically
counter := scope.Counter("requests")
```

## Best Practices

1. **Reuse metric instances**: Create metrics once and reuse them rather than creating new ones for each operation
2. **Use meaningful names**: Follow naming conventions like `<namespace>_<metric>_<unit>`
3. **Limit cardinality**: Be careful with label values to avoid metric explosion
4. **Choose appropriate types**: Use counters for totals, gauges for current values, histograms for distributions
5. **Use scopes**: Organize related metrics under common scopes
6. **Use instruments for functions**: When tracking function execution, prefer `Instrument` over manually managing counters and timers

## Performance

This library is optimized for high-performance scenarios:

- Lock-free atomic operations for counters and gauges
- Efficient label marshaling and hashing
- Minimal memory allocations
- Object pooling for timers

## License

See LICENSE file for details.
