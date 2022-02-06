package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/imroc/req"

	"github.com/rfyiamcool/go-tracer"
)

var (
	serviceName = "client"
	traceAddr   = "127.0.0.1:6831"
)

func init() {
	flag.StringVar(&traceAddr, "trace-addr", "127.0.0.1:6831", "tracer agent address")
	flag.Parse()
}

func main() {
	_, _, err := tracer.NewTracer(serviceName, traceAddr, tracer.WithQueueSize(100000), tracer.WithFlushInterval(1))
	if err != nil {
		panic(err.Error())
	}
	defer tracer.Close()

	ctx, span := tracer.StartSpanContext(tracer.GetFunc())
	defer span.Finish()

	benchmarkRequest(ctx)

	log.Printf("client request end !!!")
	time.Sleep(2 * time.Second) // wait to flush trace events.
}

func benchmarkRequest(ctx context.Context) {
	span, cctx := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer span.Finish()

	cctx = tracer.ContextWithSpan(ctx, span.SetBaggageItem("kkkk", "vvvv"))
	log.Printf("request x-trace-id  ===>  %s ", tracer.GetFullTraceID(span))

	var count = 3
	for i := 0; i < count; i++ {
		getPing(cctx)
	}
}

func getPing(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer span.Finish()

	header := http.Header{}
	err := tracer.InjectHttpHeader(span, header)
	if err != nil {
		log.Panic(err.Error())
	}

	resp, err := req.Get("http://127.0.0.1:8080/ping", header)
	if err != nil {
		span.LogKV("http.error", err.Error())
		fmt.Println("request failed, err: ", err.Error())
		return
	}

	span.LogKV("header", resp.Response().Header)
	span.LogKV("response.body", resp.String())
	log.Println("resp header ", resp.Response().Header)
}
