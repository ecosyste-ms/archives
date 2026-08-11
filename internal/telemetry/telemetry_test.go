package telemetry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	testAppName  = "Archives"
	testEndpoint = "https://collector.example.com"
	testPushKey  = "push-key"
	testRevision = "abc123"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("APPSIGNAL_APP_NAME", testAppName)
	t.Setenv("APPSIGNAL_APP_ENV", "staging")
	t.Setenv("APPSIGNAL_COLLECTOR_ENDPOINT", testEndpoint)
	t.Setenv("APPSIGNAL_PUSH_API_KEY", testPushKey)
	t.Setenv("APP_REVISION", testRevision)

	config := ConfigFromEnv()

	if config.AppName != testAppName {
		t.Errorf("AppName = %q, want Archives", config.AppName)
	}
	if config.Environment != "staging" {
		t.Errorf("Environment = %q, want staging", config.Environment)
	}
	if config.Endpoint != testEndpoint {
		t.Errorf("Endpoint = %q, want hosted collector endpoint", config.Endpoint)
	}
	if config.PushAPIKey != testPushKey {
		t.Errorf("PushAPIKey = %q, want configured key", config.PushAPIKey)
	}
	if config.Revision != testRevision {
		t.Errorf("Revision = %q, want abc123", config.Revision)
	}
	if config.ServiceName != defaultServiceName {
		t.Errorf("ServiceName = %q, want web", config.ServiceName)
	}
}

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("APPSIGNAL_APP_ENV", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_REVISION", "")

	config := ConfigFromEnv()

	if config.Environment != defaultEnvironment {
		t.Errorf("Environment = %q, want production", config.Environment)
	}
	if config.Revision != "unknown" {
		t.Errorf("Revision = %q, want unknown", config.Revision)
	}
}

func TestStartValidatesPartialConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name:   "missing push API key",
			config: Config{Endpoint: testEndpoint},
			want:   "APPSIGNAL_PUSH_API_KEY is required",
		},
		{
			name:   "app name only",
			config: Config{AppName: testAppName},
			want:   "APPSIGNAL_PUSH_API_KEY is required",
		},
		{
			name:   "missing collector endpoint",
			config: Config{PushAPIKey: testPushKey},
			want:   "APPSIGNAL_COLLECTOR_ENDPOINT is required",
		},
		{
			name: "missing app name",
			config: Config{
				Endpoint:   testEndpoint,
				PushAPIKey: testPushKey,
			},
			want: "APPSIGNAL_APP_NAME is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Start(context.Background(), tt.config)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("Start() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestResourceForAppSignal(t *testing.T) {
	appResource, err := resourceFor(Config{
		AppName:     testAppName,
		AppPath:     defaultAppPath,
		Environment: defaultEnvironment,
		Hostname:    "archives-1",
		PushAPIKey:  testPushKey,
		Revision:    testRevision,
		ServiceName: defaultServiceName,
	})
	if err != nil {
		t.Fatalf("resourceFor() error: %v", err)
	}

	wants := map[attribute.Key]string{
		"appsignal.config.name":                 testAppName,
		"appsignal.config.environment":          defaultEnvironment,
		"appsignal.config.push_api_key":         testPushKey,
		"appsignal.config.revision":             testRevision,
		"appsignal.config.language_integration": "go",
		"appsignal.config.app_path":             defaultAppPath,
		"service.name":                          defaultServiceName,
		"host.name":                             "archives-1",
	}
	for key, want := range wants {
		value, ok := appResource.Set().Value(key)
		if !ok || value.AsString() != want {
			t.Errorf("resource attribute %q = %q, want %q", key, value.AsString(), want)
		}
	}
}

func TestHTTPHandlerRecordsRouteAndErrorWithoutQueryString(t *testing.T) {
	exporter := setupTracing(t)

	var query string
	handler := HTTPHandler("GET /archive", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		RecordError(r.Context(), errors.New("archive failed"))
		http.Error(w, "failed", http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "https://archives.example.com/archive?url=https%3A%2F%2Fexample.com%2Fpackage.tgz", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if query == "" {
		t.Fatal("wrapped handler did not receive the request query string")
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("recorded %d spans, want 1", len(spans))
	}
	span := spans[0]
	if span.Name != "GET /archive" {
		t.Errorf("span name = %q, want GET /archive", span.Name)
	}
	if span.Status.Code != codes.Error {
		t.Errorf("span status = %v, want error", span.Status.Code)
	}
	if value, ok := attributeValue(span.Attributes, "http.route"); !ok || value.AsString() != "/archive" {
		t.Errorf("http.route = %q, want /archive", value.AsString())
	}
	if _, ok := attributeValue(span.Attributes, "url.query"); ok {
		t.Error("span contains the request query string")
	}

	foundException := false
	for _, event := range span.Events {
		if event.Name == "exception" {
			foundException = true
			break
		}
	}
	if !foundException {
		t.Error("span does not contain an exception event")
	}
}

func TestHTTPTransportRecordsChildSpanWithoutQueryString(t *testing.T) {
	exporter := setupTracing(t)

	var query string
	transport := HTTPTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		query = req.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    req,
		}, nil
	}))

	ctx, parent := otel.Tracer(instrumentationName).Start(context.Background(), "request")
	req := httptest.NewRequest(http.MethodGet, "https://example.com/package.tgz?token=secret", nil).WithContext(ctx)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
	parent.End()

	if query != "token=secret" {
		t.Errorf("base transport query = %q, want token=secret", query)
	}

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("recorded %d spans, want 2", len(spans))
	}
	foundClientSpan := false
	for _, span := range spans {
		if span.SpanKind != trace.SpanKindClient {
			continue
		}
		foundClientSpan = true
		value, ok := attributeValue(span.Attributes, "url.full")
		if !ok {
			t.Error("client span does not contain url.full")
			continue
		}
		if value.AsString() != "https://example.com/package.tgz" {
			t.Errorf("client span URL = %q, want URL without query string", value.AsString())
		}
	}
	if !foundClientSpan {
		t.Error("outbound request did not record a client span")
	}
}

func setupTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shut down tracer provider: %v", err)
		}
	})
	return exporter
}

func attributeValue(attributes []attribute.KeyValue, key attribute.Key) (attribute.Value, bool) {
	for _, item := range attributes {
		if item.Key == key {
			return item.Value, true
		}
	}
	return attribute.Value{}, false
}
