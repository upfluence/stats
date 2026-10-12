package prometheus

import (
	"io"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/upfluence/log"
)

const defaultInterval = 15 * time.Second

// Option configures an Exporter with custom settings.
type Option func(*exporter)

// WithInterval configures the push interval for the exporter.
// Default is 15 seconds.
func WithInterval(interval time.Duration) Option {
	return func(e *exporter) {
		e.ticker.Reset(interval)
	}
}

// WithGrouping adds a grouping label to the push gateway request.
// This is useful for distinguishing metrics from different instances or environments.
func WithGrouping(name, value string) Option {
	return func(e *exporter) {
		e.pusher = e.pusher.Grouping(name, value)
	}
}

type exporter struct {
	wg        sync.WaitGroup
	closec    chan struct{}
	closeOnce sync.Once

	ticker *time.Ticker
	pusher *push.Pusher
	logger log.Logger
}

// NewExporter creates a new Prometheus push gateway exporter that periodically
// pushes metrics to the specified gateway URL.
//
// The exporter runs in the background and pushes metrics at regular intervals.
// Call Close() to stop the exporter and perform a final push.
//
// Parameters:
//   - l: Logger for error reporting
//   - url: Push gateway URL (e.g., "http://pushgateway:9091")
//   - appName: Job name for the push gateway
//   - opts: Optional configuration (interval, grouping labels, etc.)
func NewExporter(l log.Logger, url, appName string, opts ...Option) io.Closer {
	var e = exporter{
		closec: make(chan struct{}),
		ticker: time.NewTicker(defaultInterval),
		pusher: push.New(url, appName).Gatherer(prometheus.DefaultGatherer),
		logger: l,
	}

	for _, opt := range opts {
		opt(&e)
	}

	e.wg.Add(1)
	go e.tick()

	return &e
}

func (e *exporter) push() {
	if err := e.pusher.Push(); err != nil {
		e.logger.Noticef("Push to gatherer failed: %s", err.Error())
	}
}

func (e *exporter) tick() {
	for {
		select {
		case <-e.closec:
			e.push()
			e.wg.Done()
			e.ticker.Stop()
			return
		case <-e.ticker.C:
			e.push()
		}
	}
}

func (e *exporter) Close() error {
	e.closeOnce.Do(func() { close(e.closec) })
	e.wg.Wait()

	return nil
}
