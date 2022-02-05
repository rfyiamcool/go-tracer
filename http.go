package tracer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// InjectHttpHeader
func InjectHttpHeader(span opentracing.Span, header http.Header) error {
	return gtracer.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(header))
}

// ExtractHttpHeader
func ExtractHttpHeader(header http.Header) (opentracing.SpanContext, error) {
	spctx, err := gtracer.Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(header),
	)
	return spctx, err
}

// TracingMiddleware gin middleware
func TracingMiddleware(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			serverSpan    opentracing.Span
			operationName = fmt.Sprintf("%s:%s", c.Request.URL.Path, c.Request.Method)
		)

		spctx, err := ExtractHttpHeader(c.Request.Header)
		if err != nil {
			serverSpan = gtracer.StartSpan(operationName)
		} else {
			serverSpan = opentracing.StartSpan(
				operationName,
				ext.RPCServerOption(spctx),
			)
		}

		defer serverSpan.Finish()

		c.Set("root_span_ctx", serverSpan.Context())

		// ext.Component.Set(serverSpan, name)
		serverSpan.SetTag("http.url", c.Request.URL.Path)
		serverSpan.SetTag("http.method", c.Request.Method)
		serverSpan.SetTag("http.headers.xff", c.Request.Header.Get("X-Forwarded-For"))
		serverSpan.SetTag("http.headers.ua", c.Request.Header.Get("User-Agent"))
		serverSpan.SetTag("http.request.time", time.Now().Format(time.RFC3339))
		serverSpan.SetTag("http.headers", marshal(c.Request.Header))

		body, err := ioutil.ReadAll(c.Request.Body)
		if err == nil {
			opentracing.Tag{Key: "http.request.body", Value: string(body)}.Set(serverSpan)
		}

		traceID, spanID := GetTraceSpanIDs(serverSpan)
		c.Writer.Header().Set(HeaderTraceID, traceID)
		c.Writer.Header().Set(HeaderSpanID, spanID)
		c.Request = c.Request.WithContext(opentracing.ContextWithSpan(c.Request.Context(), serverSpan))

		c.Next()

		ext.HTTPStatusCode.Set(serverSpan, uint16(c.Writer.Status()))
		serverSpan.SetTag("http.request.errors", c.Errors.String())
	}
}
