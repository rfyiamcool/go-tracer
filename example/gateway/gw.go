package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req"

	"github.com/rfyiamcool/go-tracer"
)

var (
	bindAddr    = ":8080"
	serviceName = "gateway"
	traceAddr   = "172.16.0.46:6831"
)

func main() {
	_, _, err := tracer.NewTracer(serviceName, traceAddr, tracer.WithQueueSize(100000))
	if err != nil {
		panic(err.Error())
	}
	defer tracer.Close()

	server()
}

func server() {
	r := gin.Default()
	r.Use(tracer.TracingMiddleware(serviceName))
	r.GET("/ping", handlePing)
	r.Run(bindAddr)
}

func handlePing(c *gin.Context) {
	ctx := c.Request.Context()

	// benchmark()
	handlePost(ctx)
	handleRecord(ctx)
	handleLogic(ctx, []string{"aa", "bb"})

	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func handlePost(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer span.Finish()

	fakeSleepN(300)
}

func handleRecord(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer span.Finish()

	fakeSleepN(500)
}

func handleLogic(ctx context.Context, args interface{}) (reply string) {
	span, cctx := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer func() {
		span.SetTag("request", args)
		span.SetTag("reply", map[string]interface{}{"url": "xiaorui.cc", "age": 18, "addr": "china bj"})
		span.Finish()
	}()

	fakeSleep()

	reply = time.Now().String()

	handleInsideFunc(cctx, []string{"in1", "in2"})
	return
}

func handleInsideFunc(ctx context.Context, params interface{}) error {
	var reply string

	span, cctx := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer func() {
		span.SetTag("request", params)
		span.SetTag("reply", reply)
		span.Finish()
	}()

	fakeSleep()

	resp, err := httpGet(cctx, "http://127.0.0.1:8181/ping", map[string]interface{}{"k1": "v1"})
	if err != nil {
		return err
	}

	reply = resp.String()
	return nil
}

func httpGet(ctx context.Context, url string, data map[string]interface{}) (*req.Resp, error) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer func() {
		span.SetTag("url", url)
		span.Finish()
	}()

	header := make(http.Header, 10)
	err := tracer.InjectHttpHeader(span, header)
	if err != nil {
		span.LogFields(tracer.LogString("inject-error", err.Error()))
	}

	resp, err := req.Get(url, header)
	return resp, err
}

func fakeSleep() {
	time.Sleep(time.Duration(rand.Intn(1000) * int(time.Millisecond)))
}

func fakeSleepN(n int) {
	time.Sleep(time.Duration(rand.Intn(n) * int(time.Millisecond)))
}

func benchmark() {
	var count = 5000
	for i := 0; i < count; i++ {
		i := i
		go func() {
			span := tracer.StartSpan("benchmark_test")
			defer span.Finish()

			span.SetTag("id", i)

			fakeSleepN(1000)

			if i%1000 == 0 {
				fmt.Printf("benchmark trace id: %s\n", tracer.GetTraceID(span))
			}
		}()
	}
}
