package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/spf13/cast"
	grpcotel "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

var (
	hostname, _       = os.Hostname()
	tracerProvider    *tracesdk.TracerProvider
	defaultPropagator propagation.TextMapPropagator
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
	defaultPropagator = propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// set global
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(defaultPropagator)
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

// GetPropagator
func GetPropagator() propagation.TextMapPropagator {
	return defaultPropagator
}

// Start
func Start(ctx context.Context, operation string) (context.Context, trace.Span) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, operation)
	return ctx, span
}

// StartSpan
func StartSpan(ctx context.Context, operation string) (context.Context, *Span) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, operation)
	return ctx, newSpan(ctx, span)
}

// InjectHttpHeader
func InjectHttpHeader(ctx context.Context, header http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
}

// ExtractHttpHeader
func ExtractHttpHeader(ctx context.Context, header http.Header) (context.Context, trace.Span) {
	cctx := defaultPropagator.Extract(ctx, propagation.HeaderCarrier(header))
	span := trace.SpanFromContext(cctx)
	return cctx, span
}

const (
	headerTraceparent = "traceparent"
	HeaderTracesTate  = "tracestate"
)

// SpanFromString
func SpanFromString(xid string, spanName string) (context.Context, trace.Span) {
	header := make(http.Header)
	header.Set(headerTraceparent, xid)

	ctx := defaultPropagator.Extract(context.Background(), propagation.HeaderCarrier(header))
	span := trace.SpanFromContext(ctx)
	return ctx, span
}

// ContextToString
func ContextToString(ctx context.Context) string {
	header := make(http.Header)
	InjectHttpHeader(ctx, header)
	return header.Get(headerTraceparent)
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

// GrpcDialOption grpc client option
func GrpcUnaryDialOption() grpc.DialOption {
	return grpc.WithUnaryInterceptor(UnaryClientInterceptor())
}

// GrpcUnaryServerOption grpc server option
func GrpcUnaryServerOption() grpc.ServerOption {
	return grpc.UnaryInterceptor(UnaryServerInterceptor())
}

type Span struct {
	ctx  context.Context
	span trace.Span

	once sync.Once
}

func newSpan(ctx context.Context, span trace.Span) *Span {
	return &Span{
		ctx:  ctx,
		span: span,
	}
}

func (sp *Span) DeferEnd() func() {
	return func() {
		sp.once.Do(func() {
			sp.span.End()
		})
	}
}

func (sp *Span) End() {
	sp.once.Do(func() {
		sp.span.End()
	})
}

func (sp *Span) Span() trace.Span {
	return sp.span
}

func (sp *Span) SpanContext() trace.SpanContext {
	return sp.span.SpanContext()
}

func (sp *Span) AddEvent(val interface{}) {
	sp.span.AddEvent(toString(val))
}

func (sp *Span) AddJsonEvent(val interface{}) {
	bs, _ := json.Marshal(val)
	sp.span.AddEvent(string(bs))
}

func (sp *Span) AddEventa(args ...interface{}) {
	bs, _ := json.Marshal(args)
	sp.span.AddEvent(string(bs))
}

func (sp *Span) AddEventf(format string, args ...interface{}) {
	sp.span.AddEvent(fmt.Sprintf(format, args...))
}

func (sp *Span) SetAttributes(key string, val interface{}) {
	sp.SetTag(key, val)
}

func (sp *Span) SetTag(key string, val interface{}) {
	out := toString(val)
	sp.span.SetAttributes(attribute.String(key, out))
}

func (sp *Span) SetJsonTag(key string, val interface{}) {
	bs, _ := json.Marshal(val)
	sp.span.SetAttributes(attribute.String(key, string(bs)))
}

func (sp *Span) SetStatus(code codes.Code, description string) {
	sp.span.SetStatus(code, description)
}

func (sp *Span) SetName(name string) {
	sp.span.SetName(name)
}

func (sp *Span) TraceID() {
	sp.span.TracerProvider()
}

func (sp *Span) SpanID(name string) {
	sp.span.SpanContext().SpanID()
}

func toString(val interface{}) string {
	out, err := cast.ToStringE(val)
	if err != nil {
		bs, err := json.Marshal(val)
		if err != nil {
			out = fmt.Sprintf("trace marshal failed, val: %v", val)
		} else {
			out = string(bs)
		}
	}

	return out
}
