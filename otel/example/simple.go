package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rfyiamcool/go-tracer/otel"
)

var (
	serviceNmae = "biz"
	url         string
)

func init() {
	flag.StringVar(&url, "url", "http://127.0.0.1:14268/api/traces", "input jaeger url")
	flag.Parse()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tp, err := otel.New(serviceNmae, url)
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Shutdown(context.Background())

	cctx, span := otel.Start(ctx, "main")
	defer span.End()

	foot(cctx)
	log.Printf("trace-id %s", span.SpanContext().TraceID().String())
}

func foot(ctx context.Context) {
	cctx, span := otel.Start(ctx, "foot")
	defer span.End()

	time.Sleep(300 * time.Millisecond)
	bar(cctx)
}

func bar(ctx context.Context) {
	cctx, span := otel.Start(ctx, "bar")
	defer span.End()

	time.Sleep(500 * time.Millisecond)
	parse(cctx)
}

func parse(ctx context.Context) {
	header := make(http.Header)
	header.Add("user-agent", "chrome 1.1.1")

	// inject
	otel.InjectHttpHeader(ctx, header)
	fmt.Println("inject header: ", header)

	// extract
	_, span := otel.ExtractHttpHeader(context.Background(), header)
	traceID := span.SpanContext().TraceID()
	fmt.Println("extract header: ", traceID)

	time.Sleep(100 * time.Millisecond)
}
