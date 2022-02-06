package tracer

import (
	"context"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MetadataHeader metadata Reader and Writer
type MetadataHeader struct {
	metadata.MD
}

// ForeachKey implements ForeachKey of opentracing.TextMapReader
func (c MetadataHeader) ForeachKey(handler func(key, val string) error) error {
	for k, vs := range c.MD {
		for _, v := range vs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set implements Set() of opentracing.TextMapWriter
func (c MetadataHeader) Set(key, val string) {
	key = strings.ToLower(key)
	c.MD[key] = append(c.MD[key], val)
}

// DialOption grpc client option
func GrpcDialOption() grpc.DialOption {
	return grpc.WithUnaryInterceptor(ClientInterceptor())
}

// ServerOption grpc server option
func GrpcServerOption() grpc.ServerOption {
	return grpc.UnaryInterceptor(ServerInterceptor())
}

// InjectGrpcMD
func InjectGrpcMD(span opentracing.Span, header metadata.MD, ctx context.Context) (context.Context, metadata.MD) {
	if header == nil {
		header = metadata.New(nil)
	}

	mdh := MetadataHeader{header}
	err := gtracer.Inject(span.Context(), opentracing.TextMap, mdh)
	if err != nil {
		span.LogFields(LogString("inject-error", err.Error()))
	}

	return metadata.NewOutgoingContext(ctx, header), header
}

// ExtractGrpcHeader
func ExtractGrpcHeader(ctx context.Context) (opentracing.SpanContext, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}

	spctx, err := gtracer.Extract(opentracing.TextMap, MetadataHeader{md})
	return spctx, err
}

// ClientInterceptor grpc client wrapper
func ClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string,
		req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		var parentCtx opentracing.SpanContext
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			parentCtx = parentSpan.Context()
		}

		span := gtracer.StartSpan(
			method,
			opentracing.ChildOf(parentCtx),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
			ext.SpanKindRPCClient,
		)
		defer span.Finish()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		mdWriter := MetadataHeader{md}
		err := gtracer.Inject(span.Context(), opentracing.TextMap, mdWriter)
		if err != nil {
			span.LogFields(log.String("inject-error", err.Error()))
		}

		newCtx := metadata.NewOutgoingContext(ctx, md)
		err = invoker(newCtx, method, req, reply, cc, opts...)
		if err != nil {
			span.LogFields(log.String("call-error", err.Error()))
		}
		return err
	}
}

// ServerInterceptor grpc server wrapper
func ServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if gtracer == nil {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		spanContext, err := gtracer.Extract(opentracing.TextMap, MetadataHeader{md})
		var span opentracing.Span

		if err != nil && err != opentracing.ErrSpanContextNotFound {
			span = StartSpan(info.FullMethod)
		} else {
			span = gtracer.StartSpan(
				info.FullMethod,
				ext.RPCServerOption(spanContext),
				opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
				ext.SpanKindRPCServer,
			)
		}

		defer span.Finish()
		ctx = ContextWithSpan(ctx, span)
		resp, err = handler(ctx, req)
		GrpcSendHeader(ctx, nil) // try to send header if not send header.
		return resp, err
	}
}

// GrpcSendHeader grpc.SendHeader is called only once.
func GrpcSendHeader(ctx context.Context, header metadata.MD) {
	if header == nil {
		header = metadata.Pairs()
	}
	se := GetSpanEntryFromCtx(ctx)
	if se.IsNull() {
		return
	}

	header.Append(HeaderTraceID, se.TraceID)
	header.Append(HeaderSpanID, se.SpanID)
	header.Append(HeaderXTraceID, se.String())

	grpc.SendHeader(ctx, header)
}
