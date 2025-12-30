package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor returns a gRPC interceptor that records metrics for each request.
func UnaryServerInterceptor(collector *Collector, exporter *PrometheusExporter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		method := info.FullMethod

		// Record request
		collector.RecordRequest(method)
		if exporter != nil {
			exporter.RecordRequest(method)
		}

		// Call handler
		resp, err := handler(ctx, req)

		// Record duration
		duration := time.Since(start).Seconds()
		collector.RecordDuration(method, duration)
		if exporter != nil {
			exporter.RecordDuration(method, duration)
		}

		// Record error if any
		if err != nil {
			collector.RecordError(method)
			if exporter != nil {
				exporter.RecordError(method)
			}
		}

		return resp, err
	}
}
