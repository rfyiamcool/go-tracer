package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/spf13/cast"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

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

func (sp *Span) TraceID() string {
	return sp.span.SpanContext().TraceID().String()
}

func (sp *Span) SpanID() string {
	return sp.span.SpanContext().SpanID().String()
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
