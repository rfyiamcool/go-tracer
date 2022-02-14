package otel

import (
	"context"

	grpcotel "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

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

// GrpcSendHeader insert traceID and spanID to header, grpc send header
func GrpcSendHeader(ctx context.Context, header metadata.MD) {
	if header == nil {
		header = metadata.Pairs()
	}

	spctx := SpanContextFromContext(ctx)
	header.Append(HeaderTraceID, spctx.TraceID().String())
	header.Append(HeaderSpanID, spctx.SpanID().String())

	grpc.SendHeader(ctx, header)
}
