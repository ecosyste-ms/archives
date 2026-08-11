package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultAppPath      = "/app"
	defaultEnvironment  = "production"
	defaultServiceName  = "web"
	instrumentationName = "github.com/ecosyste-ms/archives"
)

type Config struct {
	AppName     string
	AppPath     string
	Environment string
	Endpoint    string
	Hostname    string
	PushAPIKey  string
	Revision    string
	ServiceName string
}

func ConfigFromEnv() Config {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	appPath, err := os.Getwd()
	if err != nil {
		appPath = defaultAppPath
	}

	return Config{
		AppName:     os.Getenv("APPSIGNAL_APP_NAME"),
		AppPath:     appPath,
		Environment: firstSet(os.Getenv("APPSIGNAL_APP_ENV"), os.Getenv("APP_ENV"), defaultEnvironment),
		Endpoint:    firstSet(os.Getenv("APPSIGNAL_COLLECTOR_ENDPOINT"), os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		Hostname:    hostname,
		PushAPIKey:  os.Getenv("APPSIGNAL_PUSH_API_KEY"),
		Revision:    firstSet(os.Getenv("APP_REVISION"), "unknown"),
		ServiceName: defaultServiceName,
	}
}

func Start(ctx context.Context, config Config) (func(context.Context) error, error) {
	shutdown := func(context.Context) error { return nil }
	if config.AppName == "" && config.PushAPIKey == "" && config.Endpoint == "" {
		return shutdown, nil
	}
	if config.PushAPIKey == "" {
		return shutdown, errors.New("APPSIGNAL_PUSH_API_KEY is required")
	}
	if config.Endpoint == "" {
		return shutdown, errors.New("APPSIGNAL_COLLECTOR_ENDPOINT is required")
	}
	if config.AppName == "" {
		return shutdown, errors.New("APPSIGNAL_APP_NAME is required")
	}

	appResource, err := resourceFor(config)
	if err != nil {
		return shutdown, err
	}

	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(config.Endpoint))
	if err != nil {
		return shutdown, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(appResource),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}

func HTTPHandler(pattern string, next http.Handler) http.Handler {
	route := pattern
	if _, value, ok := strings.Cut(pattern, " "); ok {
		route = value
	}

	instrumented := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trace.SpanFromContext(r.Context()).SetAttributes(attribute.String("http.route", route))
		next.ServeHTTP(w, requestWithOriginalURL(r))
	}), pattern, otelhttp.WithSpanNameFormatter(func(string, *http.Request) string {
		return pattern
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		instrumented.ServeHTTP(w, requestWithoutQuery(r))
	})
}

func HTTPTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	instrumented := otelhttp.NewTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return base.RoundTrip(requestWithOriginalURL(r))
	}))

	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return instrumented.RoundTrip(requestWithoutQuery(r))
	})
}

func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, trace.WithStackTrace(true))
	span.SetStatus(codes.Error, err.Error())
}

func resourceFor(config Config) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("appsignal.config.name", config.AppName),
			attribute.String("appsignal.config.environment", config.Environment),
			attribute.String("appsignal.config.push_api_key", config.PushAPIKey),
			attribute.String("appsignal.config.revision", config.Revision),
			attribute.String("appsignal.config.language_integration", "go"),
			attribute.String("appsignal.config.app_path", config.AppPath),
			attribute.StringSlice("appsignal.config.filter_attributes", []string{"url.query"}),
			attribute.String("service.name", config.ServiceName),
			attribute.String("host.name", config.Hostname),
		),
	)
}

func firstSet(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type requestURLKey struct{}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func requestWithoutQuery(r *http.Request) *http.Request {
	originalURL := *r.URL
	request := r.Clone(context.WithValue(r.Context(), requestURLKey{}, &originalURL))
	requestURL := *r.URL
	requestURL.RawQuery = ""
	request.URL = &requestURL
	return request
}

func requestWithOriginalURL(r *http.Request) *http.Request {
	originalURL, ok := r.Context().Value(requestURLKey{}).(*url.URL)
	if !ok {
		return r
	}

	request := r.Clone(r.Context())
	request.URL = originalURL
	return request
}
