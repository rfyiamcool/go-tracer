package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/rfyiamcool/grpc-example/simple/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rfyiamcool/go-tracer/otel"
)

var (
	bindAddr    = ":8181"
	serviceName = "service"
	url         string

	gclient pb.UserServiceClient
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

	initGrpcClient()

	server()
}

func initGrpcClient() {
	conn, err := grpc.Dial("127.0.0.1:3001", grpc.WithInsecure(), grpc.WithUnaryInterceptor(otel.UnaryClientInterceptor()))
	if err != nil {
		log.Fatalf("connect server error: %v", err)
	}

	// init grpc client
	gclient = pb.NewUserServiceClient(conn)
}

func server() {
	r := gin.Default()
	r.Use(otel.GinMiddleware(serviceName))
	r.GET("/ping", handlePing)
	r.Run(bindAddr)
}

func handlePing(c *gin.Context) {
	ctx := c.Request.Context()

	hanldePost(ctx)
	handleCall(ctx, []string{"aa", "bb"})

	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func hanldePost(ctx context.Context) {
	cctx, span := otel.StartSpan(ctx, "post")
	defer span.End()

	handleGrpcService(cctx, 1111)
	handleGrpcService(cctx, 2222)
}

func handleGrpcService(ctx context.Context, uid int) {
	cctx, span := otel.StartSpan(ctx, "handleGrpcService")
	defer span.End()

	// inject header, fill biz header
	md := metadata.Pairs("key-111", "val-111")
	md.Set("key-222", "val-222")

	// request
	var rheader metadata.MD
	resp, err := gclient.GetUserInfo(
		cctx,
		&pb.UserRequest{ID: int32(uid)},
		grpc.Header(&rheader),
	)

	// tag
	span.SetAttributes("method", "GetUserInfo")

	// log
	span.AddEventa("resp.body", md)
	span.AddEventa("resp.body", resp)
	span.AddEventa("resp.error", err)

	if err != nil {
		fmt.Println("grpc request failed, err: ", err.Error())
		return
	}
}

func handleCall(ctx context.Context, args interface{}) (reply string) {
	cctx, span := otel.Start(ctx, "handle_call")
	defer span.End()

	fakeSleep()

	handleInsideFunc(cctx, []string{"in1", "in2"})
	return time.Now().String()
}

func handleInsideFunc(ctx context.Context, params interface{}) (reply string) {
	cctx, span := otel.Start(ctx, "handle_call")
	defer span.End()

	fakeSleep()

	handleRedis(cctx)
	handleDatabase(cctx)

	return time.Now().String()
}

func handleRedis(ctx context.Context) {
	getUserFromRedis(ctx)
	updateDataFromRedis(ctx)
}

func getUserFromRedis(ctx context.Context) {
	_, span := otel.Start(ctx, "handle_call")
	defer span.End()

	fakeSleepN(500)
	return
}

func updateDataFromRedis(ctx context.Context) {
	_, span := otel.Start(ctx, "handle_call")
	defer span.End()

	fakeSleepN(500)
	return
}

func handleDatabase(ctx context.Context) {
	getUserFromDB(ctx)
	updateDataFromDB(ctx)
}

func getUserFromDB(ctx context.Context) {
	_, span := otel.Start(ctx, "handle_call")
	defer span.End()

	fakeSleepN(500)
	return
}

func updateDataFromDB(ctx context.Context) {
	_, span := otel.Start(ctx, "handle_call")
	defer span.End()

	fakeSleepN(500)
	return
}

func fakeSleep() {
	time.Sleep(time.Duration(rand.Intn(1000) * int(time.Millisecond)))
}

func fakeSleepN(n int) {
	time.Sleep(time.Duration(rand.Intn(n) * int(time.Millisecond)))
}
