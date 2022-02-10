package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/imroc/req"

	"github.com/rfyiamcool/go-tracer/otel"
)

var (
	bindAddr    = ":8080"
	serviceName = "client"
	url         string
)

func init() {
	flag.StringVar(&url, "url", "http://127.0.0.1:14268/api/traces", "input jaeger url")
	flag.Parse()
}

func main() {
	tp, err := otel.New(serviceName, otel.WithMode(otel.ModeCollectorHttp), otel.WithAddress(url), otel.WithQueueSize(3000))
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Shutdown(context.Background())

	ctx, span := otel.StartSpan(context.Background(), "main")
	defer span.End()

	benchmarkRequest(ctx)

	log.Println("client request end !!!")
	time.Sleep(2 * time.Second) // wait to flush trace events.
}

func benchmarkRequest(ctx context.Context) {
	cctx, span := otel.StartSpan(ctx, "benchmarkRequest")
	defer span.End()

	log.Printf("request x-trace-id  ===>  %s ", span.TraceID())

	var count = 3
	for i := 0; i < count; i++ {
		getPing(cctx)
	}
}

func getPing(ctx context.Context) {
	cctx, span := otel.StartSpan(ctx, "getPing")
	defer span.End()

	header := http.Header{}
	otel.InjectHttpHeader(cctx, header)

	resp, err := req.Get("http://127.0.0.1:8080/ping", header)
	if err != nil {
		log.Printf("request failed, err: %s", err.Error())
		return
	}

	log.Printf("resp header %v", resp.Response().Header)
}
