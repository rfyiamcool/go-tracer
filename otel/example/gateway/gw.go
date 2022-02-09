package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/rfyiamcool/go-tracer/otel"
)

var (
	bindAddr    = ":8080"
	serviceName = "gateway"
	url         string
)

func init() {
	flag.StringVar(&url, "url", "http://127.0.0.1:14268/api/traces", "input jaeger url")
	flag.Parse()
}

func main() {
	tp, err := otel.New(serviceName, url)
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Shutdown(context.Background())

	server()
}

func server() {
	r := gin.Default()
	r.Use(otel.GinMiddleware(serviceName))
	r.GET("/ping", handlePing)
	r.Run(bindAddr)
}

func handlePing(c *gin.Context) {
	ctx := c.Request.Context()

	handleRecord(ctx)
	handleLogic(ctx, []string{"aa", "bb"})

	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func handleRecord(ctx context.Context) {
	_, span := otel.StartSpan(ctx, "http_get")
	defer span.End()

	// jaeger log
	span.AddEvent("this is handleRecord function")

	// jaeger tag
	span.SetTag("record", "risk")

	fakeSleepN(500)
}

func handleLogic(ctx context.Context, args interface{}) error {
	cctx, span := otel.StartSpan(ctx, "logic")
	defer span.End()

	_, err := httpGet(cctx, "http://127.0.0.1:8181/ping", map[string]interface{}{"k1": "v1"})
	fakeSleepN(100)
	return err
}

func httpGet(ctx context.Context, url string, data map[string]interface{}) (*req.Resp, error) {
	cctx, span := otel.StartSpan(ctx, "http_get")
	defer span.End()

	// jaeger log
	span.AddEvent("this is first log")
	span.AddEvent("this is second log")

	// jaeger tag
	span.SetAttributes("url", url)
	span.SetAttributes("data", data)
	span.SetJsonTag("data1", data)

	header := make(http.Header, 10)
	otel.InjectHttpHeader(cctx, header)
	resp, err := req.Get(url, header)
	return resp, err
}

func fakeSleep() {
	time.Sleep(time.Duration(rand.Intn(1000) * int(time.Millisecond)))
}

func fakeSleepN(n int) {
	time.Sleep(time.Duration(rand.Intn(n) * int(time.Millisecond)))
}
