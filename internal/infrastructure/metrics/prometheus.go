package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusExporter exports metrics to Prometheus format.
type PrometheusExporter struct {
	collector *Collector

	// Prometheus metrics
	cacheHits        prometheus.Counter
	cacheMisses      prometheus.Counter
	cacheHitRate     prometheus.Gauge
	cacheKeys        prometheus.Gauge
	cacheMemoryBytes prometheus.Gauge
	cacheEvictions   prometheus.Counter
	grpcRequests     *prometheus.CounterVec
	grpcDuration     *prometheus.HistogramVec
	grpcErrors       *prometheus.CounterVec

	// Last known cumulative values for delta calculation
	lastHits      uint64
	lastMisses    uint64
	lastEvictions uint64
}

// NewPrometheusExporter creates a new Prometheus exporter.
// registry allows passing a custom prometheus.Registerer (pass nil for default).
func NewPrometheusExporter(collector *Collector, registry prometheus.Registerer) *PrometheusExporter {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}
	factory := promauto.With(registry)
	return &PrometheusExporter{
		collector: collector,
		cacheHits: factory.NewCounter(prometheus.CounterOpts{
			Name: "keruberosu_check_cache_hits_total",
			Help: "Total number of cache hits for permission checks",
		}),
		cacheMisses: factory.NewCounter(prometheus.CounterOpts{
			Name: "keruberosu_check_cache_misses_total",
			Help: "Total number of cache misses for permission checks",
		}),
		cacheHitRate: factory.NewGauge(prometheus.GaugeOpts{
			Name: "keruberosu_check_cache_hit_rate",
			Help: "Current cache hit rate (0.0 to 1.0)",
		}),
		cacheKeys: factory.NewGauge(prometheus.GaugeOpts{
			Name: "keruberosu_check_cache_keys_current",
			Help: "Current number of keys in the check cache",
		}),
		cacheMemoryBytes: factory.NewGauge(prometheus.GaugeOpts{
			Name: "keruberosu_check_cache_memory_bytes",
			Help: "Current memory usage of the check cache in bytes",
		}),
		cacheEvictions: factory.NewCounter(prometheus.CounterOpts{
			Name: "keruberosu_check_cache_evictions_total",
			Help: "Total number of cache evictions due to memory limits",
		}),
		grpcRequests: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "keruberosu_grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method"},
		),
		grpcDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "keruberosu_grpc_request_duration_seconds",
				Help:    "Duration of gRPC requests in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0},
			},
			[]string{"method"},
		),
		grpcErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "keruberosu_grpc_errors_total",
				Help: "Total number of gRPC errors",
			},
			[]string{"method"},
		),
	}
}

// Update updates Gauge metrics from the collector.
// Counters are updated via interceptor, so only update gauges here.
// This should be called periodically (e.g., every 10 seconds).
func (e *PrometheusExporter) Update() {
	cacheMetrics := e.collector.GetCacheMetrics()
	e.cacheHitRate.Set(cacheMetrics.HitRate)
	e.cacheKeys.Set(float64(cacheMetrics.KeysCurrent))
	e.cacheMemoryBytes.Set(float64(cacheMetrics.MemoryBytes))

	// Update counters from collector's cumulative values (delta calculation)
	currentHits := cacheMetrics.Hits
	if currentHits > e.lastHits {
		e.cacheHits.Add(float64(currentHits - e.lastHits))
		e.lastHits = currentHits
	}
	currentMisses := cacheMetrics.Misses
	if currentMisses > e.lastMisses {
		e.cacheMisses.Add(float64(currentMisses - e.lastMisses))
		e.lastMisses = currentMisses
	}
	currentEvictions := cacheMetrics.Evictions
	if currentEvictions > e.lastEvictions {
		e.cacheEvictions.Add(float64(currentEvictions - e.lastEvictions))
		e.lastEvictions = currentEvictions
	}
}

// RecordRequest records a request in Prometheus.
func (e *PrometheusExporter) RecordRequest(method string) {
	e.grpcRequests.WithLabelValues(method).Inc()
}

// RecordDuration records a duration in Prometheus.
func (e *PrometheusExporter) RecordDuration(method string, durationSeconds float64) {
	e.grpcDuration.WithLabelValues(method).Observe(durationSeconds)
}

// RecordError records an error in Prometheus.
func (e *PrometheusExporter) RecordError(method string) {
	e.grpcErrors.WithLabelValues(method).Inc()
}

// RecordCacheHit records a cache hit.
func (e *PrometheusExporter) RecordCacheHit() {
	e.cacheHits.Inc()
}

// RecordCacheMiss records a cache miss.
func (e *PrometheusExporter) RecordCacheMiss() {
	e.cacheMisses.Inc()
}

// RecordCacheEviction records a cache eviction.
func (e *PrometheusExporter) RecordCacheEviction() {
	e.cacheEvictions.Inc()
}
