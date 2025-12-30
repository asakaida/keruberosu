package metrics

import (
	"context"
	"errors"
	"sync"
	"testing"

	"google.golang.org/grpc"
)

// testExporter is a shared exporter instance for all tests to avoid
// duplicate Prometheus metric registration errors.
var (
	testExporter     *PrometheusExporter
	testExporterOnce sync.Once
)

func getTestExporter(collector *Collector) *PrometheusExporter {
	testExporterOnce.Do(func() {
		testExporter = NewPrometheusExporter(collector)
	})
	return testExporter
}

func TestUnaryServerInterceptor_RecordsRequest(t *testing.T) {
	collector := NewCollector()

	interceptor := UnaryServerInterceptor(collector, nil)

	// Create mock handler that succeeds
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Call interceptor
	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that request was recorded
	apiMetrics := collector.GetAPIMetrics()
	if count, ok := apiMetrics.RequestCounts["/test.Service/TestMethod"]; !ok || count != 1 {
		t.Errorf("expected request count 1 for /test.Service/TestMethod, got %d", count)
	}
}

func TestUnaryServerInterceptor_RecordsDuration(t *testing.T) {
	collector := NewCollector()

	interceptor := UnaryServerInterceptor(collector, nil)

	// Create mock handler that succeeds
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/DurationMethod",
	}

	// Call interceptor
	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that duration was recorded (should be > 0)
	apiMetrics := collector.GetAPIMetrics()
	if _, ok := apiMetrics.TotalDurationSeconds["/test.Service/DurationMethod"]; !ok {
		t.Error("expected duration to be recorded for /test.Service/DurationMethod")
	}
}

func TestUnaryServerInterceptor_RecordsError(t *testing.T) {
	collector := NewCollector()

	interceptor := UnaryServerInterceptor(collector, nil)

	// Create mock handler that returns an error
	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/ErrorMethod",
	}

	// Call interceptor
	_, err := interceptor(context.Background(), "request", info, handler)
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	// Check that error was recorded
	apiMetrics := collector.GetAPIMetrics()
	if count, ok := apiMetrics.ErrorCounts["/test.Service/ErrorMethod"]; !ok || count != 1 {
		t.Errorf("expected error count 1 for /test.Service/ErrorMethod, got %d", count)
	}
}

func TestUnaryServerInterceptor_NoErrorNotRecorded(t *testing.T) {
	collector := NewCollector()

	interceptor := UnaryServerInterceptor(collector, nil)

	// Create mock handler that succeeds
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/SuccessMethod",
	}

	// Call interceptor
	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that no error was recorded
	apiMetrics := collector.GetAPIMetrics()
	if count, ok := apiMetrics.ErrorCounts["/test.Service/SuccessMethod"]; ok && count > 0 {
		t.Errorf("expected no error count for /test.Service/SuccessMethod, got %d", count)
	}
}

func TestUnaryServerInterceptor_NilExporter(t *testing.T) {
	collector := NewCollector()

	// Create interceptor with nil exporter
	interceptor := UnaryServerInterceptor(collector, nil)

	// Create mock handler that succeeds
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/NilExporterMethod",
	}

	// Call interceptor - should not panic with nil exporter
	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Collector should still record
	apiMetrics := collector.GetAPIMetrics()
	if count, ok := apiMetrics.RequestCounts["/test.Service/NilExporterMethod"]; !ok || count != 1 {
		t.Errorf("expected request count 1, got %d", count)
	}
}

func TestUnaryServerInterceptor_MultipleRequests(t *testing.T) {
	collector := NewCollector()

	interceptor := UnaryServerInterceptor(collector, nil)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/MultiMethod",
	}

	// Call interceptor multiple times
	for i := 0; i < 5; i++ {
		_, err := interceptor(context.Background(), "request", info, handler)
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i, err)
		}
	}

	// Check that all requests were recorded
	apiMetrics := collector.GetAPIMetrics()
	if count, ok := apiMetrics.RequestCounts["/test.Service/MultiMethod"]; !ok || count != 5 {
		t.Errorf("expected request count 5, got %d", count)
	}
}

func TestUnaryServerInterceptor_WithPrometheusExporter(t *testing.T) {
	collector := NewCollector()
	exporter := getTestExporter(collector)

	interceptor := UnaryServerInterceptor(collector, exporter)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/PrometheusMethod",
	}

	// Call interceptor
	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify collector recorded the request
	apiMetrics := collector.GetAPIMetrics()
	if count, ok := apiMetrics.RequestCounts["/test.Service/PrometheusMethod"]; !ok || count != 1 {
		t.Errorf("expected request count 1, got %d", count)
	}
}
