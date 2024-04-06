package metrics

// Prometheus metrics

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsConfig struct {
	Port   int
	Enable bool
	Model  string
	Path   string
}

type Metrics struct {
	config    MetricsConfig
	started   prometheus.Counter
	completed prometheus.Counter
	failed    prometheus.Counter
}

type MetricsApi interface {
	// Expose the metrics endpoint for the prometheus scraper
	Expose()
	DownloadStarted()
	DownloadCompleted()
	DownloadFailed()
}

func NewMetrics(config MetricsConfig) MetricsApi {
	return &Metrics{
		config: config,
	}
}

func (m *Metrics) Expose() {
	if m.config.Enable {
		go func() {
			port := fmt.Sprintf(":%d", m.config.Port)
			slog.Info("exposing metrics", "port", port)
			// register the metrics
			m.registerMetrics()
			// endpoint
			http.Handle(m.config.Path, promhttp.Handler())
			if err := http.ListenAndServe(port, nil); err != nil {
				slog.Error("failed to expose metrics", "error", err)
			}
		}()
	}
}

func (m *Metrics) DownloadStarted() {
	m.started.Inc()
}

func (m *Metrics) DownloadCompleted() {
	m.started.Inc()
}

func (m *Metrics) DownloadFailed() {
	m.failed.Inc()
}

func (m *Metrics) registerMetrics() {
	m.started = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "downloads_started",
		Help: "The total number of downloads started",
	})
	m.completed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "downloads_completed",
		Help: "The total number of downloads completed",
	})
	m.failed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "downloads_failed",
		Help: "The total number of downloads failed",
	})
}
