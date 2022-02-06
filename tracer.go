package tracer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	tracelog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-client-go/transport"
	"github.com/uber/jaeger-lib/metrics"
)

const (
	HeaderTraceID = "trace-id"
	HeaderSpanID  = "span-id"

	HeaderXTraceID = "x-trace-id"
)

var (
	gtracer opentracing.Tracer
	closer  io.Closer

	mutex sync.Mutex
)

var (
	// opentrace log
	LogString = tracelog.String
	LogInt    = tracelog.Int
	LogObject = tracelog.Object
)

const (
	protoUdp = iota
	protoHttp
)

// GetTracer
func GetTracer() opentracing.Tracer {
	return gtracer
}

// SetTracer
func SeteTracer(otracer opentracing.Tracer) {
	gtracer = otracer
	opentracing.SetGlobalTracer(otracer)
}

// Close
func Close() error {
	if closer != nil {
		return nil
	}

	return closer.Close()
}

func NewHttpSender(url string) jaeger.Transport {
	var (
		client = &http.Client{}
		trans  = &http.Transport{
			MaxIdleConnsPerHost: 50,
			MaxIdleConns:        100,
		}
	)

	client.Transport = trans
	return transport.NewHTTPTransport(
		url,
		transport.HTTPBatchSize(100),
		transport.HTTPTimeout(1*time.Second),
		transport.HTTPRoundTripper(trans),
	)
}

type Config struct {
	ServiceName         string `yaml:"service_name"`
	Address             string `yaml:"addr"`
	QueueSize           int    `yaml:"queue_size"`
	BufferFlushInterval int    `yaml:"buffer_flush_interval"`
	MaxTagLength        int    `yaml:"max_tag_length"`
	ProtoKind           int    `yaml:"proto_kind"`
}

type Option struct {
	samplerConfig  *jaegercfg.SamplerConfig
	reporterConfig *jaegercfg.ReporterConfig
	sender         jaeger.Transport

	queueSize           int
	bufferFlushInterval int
	maxTagLength        int
	protoKind           int
}

func defaultOption() *Option {
	return &Option{
		queueSize:           10000, // size of buffer queue wait to send
		maxTagLength:        256,   // default value in jaeger client
		bufferFlushInterval: 1,     // unit: time ms
	}
}

type optionFunc func(*Option) error

// WithSender
func WithSender(sender jaeger.Transport) optionFunc {
	return func(o *Option) error {
		o.sender = sender
		return nil
	}
}

// WithProtoKind
func WithProtoKind(kind int) optionFunc {
	return func(o *Option) error {
		o.protoKind = kind
		return nil
	}
}

// WithQueueSize queue size, defualt: 10000
func WithQueueSize(size int) optionFunc {
	return func(o *Option) error {
		def := 10000
		if size <= 0 || size > def {
			size = def
		}
		o.queueSize = size
		return nil
	}
}

// WithFlushInterval
func WithFlushInterval(ms int) optionFunc {
	return func(o *Option) error {
		var (
			maxThold = 5000 // 5s
			minThold = 200  // 200ms
		)

		if ms > maxThold {
			ms = maxThold
		}
		if ms < minThold {
			ms = minThold
		}
		o.bufferFlushInterval = ms
		return nil
	}
}

// WithMaxTagLength
func WithMaxTagLength(size int) optionFunc {
	return func(o *Option) error {
		def := 256
		if size <= 0 || size > def {
			size = def
		}
		o.maxTagLength = size
		return nil
	}
}

// WithSampler
func WithSampler(sampler *jaegercfg.SamplerConfig) optionFunc {
	return func(o *Option) error {
		o.samplerConfig = sampler
		return nil
	}
}

// WithReporter
func WithReporter(cfg *jaegercfg.ReporterConfig) optionFunc {
	return func(o *Option) error {
		o.reporterConfig = cfg
		return nil
	}
}

// NewTracerWithConfig
func NewTracerWithConfig(cfg *Config) (opentracing.Tracer, io.Closer, error) {
	return NewTracer(
		cfg.ServiceName,
		cfg.Address,
		WithFlushInterval(cfg.BufferFlushInterval),
		WithMaxTagLength(cfg.MaxTagLength),
		WithQueueSize(cfg.QueueSize),
		WithProtoKind(cfg.ProtoKind),
	)
}

// NewTracer
func NewTracer(serviceName string, addr string, fns ...optionFunc) (opentracing.Tracer, io.Closer, error) {
	if serviceName == "" {
		return nil, nil, errors.New("invalid service name")
	}
	if addr == "" {
		return nil, nil, errors.New("invalid address")
	}

	option := defaultOption()
	for _, fn := range fns {
		err := fn(option)
		if err != nil {
			return nil, nil, err
		}
	}

	if option.protoKind == protoUdp {
		cols := strings.Split(addr, ":")
		if len(cols) != 2 {
			return nil, nil, errors.New("invalid addr")
		}
		if !govalidator.IsIP(cols[0]) || !govalidator.IsPort(cols[1]) {
			return nil, nil, errors.New("invalid addr")
		}
	}

	// default config
	cfg := &jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler:     option.samplerConfig, // doc: https://www.jaegertracing.io/docs/1.28/sampling/
		Reporter:    option.reporterConfig,
		Headers: &jaeger.HeadersConfig{
			JaegerDebugHeader:        "x-debug-id",
			JaegerBaggageHeader:      "x-baggage",
			TraceContextHeaderName:   "x-trace-id",
			TraceBaggageHeaderPrefix: "x-ctx-",
		},
	}

	if option.samplerConfig == nil {
		cfg.Sampler = &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		}
	}
	if option.reporterConfig != nil {
		cfg.Reporter = &jaegercfg.ReporterConfig{
			LogSpans:                   true,
			QueueSize:                  option.queueSize,
			DisableAttemptReconnecting: false,
			BufferFlushInterval:        time.Duration(option.bufferFlushInterval) * time.Millisecond,
		}
	}

	sender, err := jaeger.NewUDPTransport(addr, 0)
	if err != nil {
		return nil, nil, err
	}

	if option.sender != nil {
		sender = option.sender
	}

	reporter := jaeger.NewRemoteReporter(sender)
	logger := jaegerlog.StdLogger
	jmetrics := metrics.NullFactory

	// init tracer with a logger and a metrics factory
	gtracer, closer, err = cfg.NewTracer(
		jaegercfg.Reporter(reporter),
		jaegercfg.Logger(logger),
		jaegercfg.Metrics(jmetrics),
		jaegercfg.MaxTagValueLength(option.maxTagLength),
	)

	opentracing.SetGlobalTracer(gtracer)
	return gtracer, closer, err
}

// ContextWithSpan returns a new `context.Context` that holds a reference to
// the span. If span is nil, a new context without an active span is returned.
func ContextWithSpan(ctx context.Context, span opentracing.Span) context.Context {
	return opentracing.ContextWithSpan(ctx, span)
}

// StartSpan Create, start, and return a new Span with the given `operationName`.
func StartSpan(operation string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return gtracer.StartSpan(operation, opts...)
}

// StartSpanContext
func StartSpanContext(operation string, opts ...opentracing.StartSpanOption) (context.Context, opentracing.Span) {
	span := gtracer.StartSpan(operation, opts...)
	ctx := ContextWithSpan(context.Background(), span)
	return ctx, span
}

// StartSpanFromContext starts and returns a Span with `operationName`, using
// any Span found within `ctx` as a ChildOfRef.
func StartSpanFromContext(ctx context.Context, operation string) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContext(ctx, operation)
}

// SpanFromContextExt
func StartSpanFromContextExt(ctx context.Context, fname string, request, response interface{}) (opentracing.Span, context.Context, func()) {
	span, cctx := opentracing.StartSpanFromContext(ctx, fname)
	defered := func() {
		if request != nil {
			span.SetTag("request", request)
		}
		if response != nil {
			span.SetTag("response", response)
		}
		span.Finish()
	}
	return span, cctx, defered
}

func StartSpanFromGinContext(ctx *gin.Context, fname string, request, response interface{}) (opentracing.Span, context.Context, func()) {
	return StartSpanFromContextExt(ctx.Request.Context(), fname, request, response)
}

// SpanFromContext
func SpanFromContext(ctx context.Context) opentracing.Span {
	return opentracing.SpanFromContext(ctx)
}

// GetTraceID get trace id from span
func GetTraceID(span opentracing.Span) string {
	sc, ok := span.Context().(jaeger.SpanContext)
	if ok {
		return sc.TraceID().String()
	}

	return ""
}

type SpanEntry struct {
	jaegerSpanCtx jaeger.SpanContext

	TraceID      string
	SpanID       string
	ParentSpanID string
	Flags        string
}

func (se *SpanEntry) String() string {
	return se.jaegerSpanCtx.String()
}

func (se *SpanEntry) IsNull() bool {
	if se.TraceID == "" {
		return true
	}

	return false
}

// GetSpanEntryFromCtx
func GetSpanEntryFromCtx(ctx context.Context) SpanEntry {
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return SpanEntry{}
	}

	sc, ok := span.Context().(jaeger.SpanContext)
	if ok {
		se := SpanEntry{
			jaegerSpanCtx: sc,
			TraceID:       sc.TraceID().String(),
			SpanID:        sc.SpanID().String(),
			ParentSpanID:  sc.ParentID().String(),
			Flags:         string(sc.Flags()),
		}
		return se

	}
	return SpanEntry{}
}

// GetTraceIDFromCtx get trace id from context
func GetTraceIDFromCtx(ctx context.Context) string {
	span := opentracing.SpanFromContext(ctx)
	return GetTraceID(span)
}

// GetSpanIDFromCtx get span id from context
func GetSpanIDFromCtx(ctx context.Context) string {
	span := opentracing.SpanFromContext(ctx)
	return GetSpanID(span)
}

func GetTraceSpanIDsFromCtx(ctx context.Context) (string, string) {
	span := opentracing.SpanFromContext(ctx)
	return GetTraceSpanIDs(span)
}

// GetSpanID get span id
func GetSpanID(span opentracing.Span) string {
	sc, ok := span.Context().(jaeger.SpanContext)
	if ok {
		return sc.SpanID().String()
	}

	return ""
}

// GetTraceSpanIDs
func GetTraceSpanIDs(span opentracing.Span) (string, string) {
	sc, ok := span.Context().(jaeger.SpanContext)
	if ok {
		return sc.TraceID().String(), sc.SpanID().String()
	}

	return "", ""
}

// GetParentID get parent span id
func GetParentID(span opentracing.Span) string {
	sc, ok := span.Context().(jaeger.SpanContext)
	if ok {
		return sc.ParentID().String()
	}

	return ""
}

// GetFullTraceID
func GetFullTraceID(span opentracing.Span) string {
	return GetXTraceID(span)
}

// GetXTraceID get x-trace-id by span
func GetXTraceID(span opentracing.Span) string {
	sc, ok := span.Context().(jaeger.SpanContext)
	if ok {
		// hex16(traceid):hex16(spanid):hex16(parentid):hex16(flag)
		return sc.String()
	}

	return ""
}

// MakeFullTraceID return x-trace-id
func MakeFullTraceID() string {
	span, _ := opentracing.StartSpanFromContext(context.Background(), "")
	return GetFullTraceID(span)
}

// ContextFromString build context by x-trace-id
func ContextFromString(xid string) (jaeger.SpanContext, error) {
	ctx, err := jaeger.ContextFromString(xid)
	return ctx, err
}

// ContextFromString build span by x-trace-id
func SpanFromString(operation, xid string) (opentracing.Span, error) {
	ctx, err := ContextFromString(xid)
	if err != nil {
		return nil, err
	}

	span := opentracing.StartSpan(
		operation,
		ext.RPCServerOption(ctx),
	)
	return span, err
}

func marshal(in interface{}) string {
	bs, err := json.Marshal(in)
	if err != nil {
		return fmt.Sprintf("%v", in)
	}
	return string(bs)
}
