package otel

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	ginotel "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	grpcotel "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

var (
	hostname, _    = os.Hostname()
	tracerProvider *tracesdk.TracerProvider
)

// New
func New(serviceName string, url string) (*tracesdk.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}

	tracerProvider = tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("hostname", hostname),
		)),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tracerProvider, nil
}

// Shutdown
func Shutdown(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if err := GetProvider().Shutdown(cctx); err != nil {
		log.Fatal(err)
	}
}

// GetTracer
func GetTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer("")
}

// GetProvider
func GetProvider() *tracesdk.TracerProvider {
	return tracerProvider
}

// Start
func Start(ctx context.Context, operation string) (context.Context, trace.Span) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, operation)
	return ctx, span
}

// InjectHttpHeader
func InjectHttpHeader(ctx context.Context, header http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
}

// ExtractHttpHeader
func ExtractHttpHeader(ctx context.Context, header http.Header) trace.Span {
	return trace.SpanFromContext(otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(header)))
}

// GinMiddleware
func GinMiddleware(serviceName string) gin.HandlerFunc {
	return ginotel.Middleware(serviceName)
}

// StreamClientInterceptor for grpc
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return grpcotel.StreamClientInterceptor()
}

// StreamServerInterceptor for grpc
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return grpcotel.StreamServerInterceptor()
}

// UnaryClientInterceptor for grpc
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return grpcotel.UnaryClientInterceptor()
}

// UnaryServerInterceptor for grpc
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return grpcotel.UnaryServerInterceptor()
}
