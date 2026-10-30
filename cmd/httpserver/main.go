package main

import (
	"context"
	"httpServer/internal/http1"
	"httpServer/internal/middleware"
	"httpServer/internal/mux"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	oteltrace "go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer       trace.Tracer
	otlpEndpoint string
)

const addr = ":8080"

func main() {
	ctx := context.Background()

	otlpEndpoint = os.Getenv("VT_TRACES_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "http://vtsingle-my-victoria-metrics-k8s-stack.default.svc.cluster.local:10428/insert/opentelemetry/v1/traces"
	}

	exp, err := newOTLPExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	tp := newTraceProvider(exp)
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	otel.SetTracerProvider(tp)

	tracer = tp.Tracer("httpserver")

	mux := mux.NewMyHandler()
	mux.HandleFunc("/hello", middleware.LogRequests(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "HTTP /hello")
		defer span.End()

		time.Sleep(1 * time.Second)
		doWork(ctx)
		body := []byte("hello")
		w.Write(body)
	}))

	mux.HandleFunc("/", middleware.LogRequests(func(w http.ResponseWriter, r *http.Request) {
		body := []byte("default")
		w.Write(body)
	}))

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	})

	tracedHandler := otelhttp.NewHandler(mux, "http-server")

	if err := http1.ListenAndServe(addr, tracedHandler); err != nil {
		panic(err)
	}
}

func doWork(ctx context.Context) {
	_, span := tracer.Start(ctx, "WORK")
	defer span.End()
	time.Sleep(2 * time.Second)
}

func newOTLPExporter(ctx context.Context) (oteltrace.SpanExporter, error) {
	insecureOpt := otlptracehttp.WithInsecure()
	endpointOpt := otlptracehttp.WithEndpointURL(otlpEndpoint)
	return otlptracehttp.New(ctx, insecureOpt, endpointOpt)
}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceNameKey.String("myapp"),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}
