package tracer

import (
	"context"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type Span struct {
	span opentracing.Span
	once sync.Once

	TraceID      string
	SpanID       string
	ParentSpanID string
}

func Start(ctx context.Context, operation string) (context.Context, *Span) {
	return new(Span).Start(ctx, operation)
}

func (sp *Span) Start(ctx context.Context, operation string) (context.Context, *Span) {
	span, cctx := opentracing.StartSpanFromContext(ctx, operation)
	entry := GetSpanEntryFromCtx(cctx)
	spp := &Span{
		span:         span,
		TraceID:      entry.TraceID,
		SpanID:       entry.SpanID,
		ParentSpanID: entry.ParentSpanID,
	}
	return cctx, spp
}

func (sp *Span) End() {
	sp.once.Do(func() {
		sp.span.Finish()
	})
}

func (sp *Span) SetTag(key string, val interface{}) *Span {
	sp.span.SetTag(key, val)
	return sp
}

func (sp *Span) LogFields(fields ...log.Field) *Span {
	sp.span.LogFields(fields...)
	return sp
}

func (sp *Span) LogKV(key string, val interface{}) *Span {
	sp.span.LogKV(key, val)
	return sp
}

func (sp *Span) LogString(key, val string) *Span {
	sp.span.LogFields(log.String(key, val))
	return sp
}

func (sp *Span) LogObject(key string, obj interface{}) *Span {
	sp.span.LogFields(log.Object(key, obj))
	return sp
}

func (sp *Span) LogError(err error) *Span {
	sp.span.LogFields(log.Error(err))
	return sp
}
