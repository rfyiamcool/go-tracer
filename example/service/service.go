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

	"github.com/rfyiamcool/go-tracer"
)

var (
	serviceName = "service"

	gclient   pb.UserServiceClient
	traceAddr string
	bindAddr  string
)

func init() {
	flag.StringVar(&traceAddr, "trace-addr", "127.0.0.1:6831", "tracer agent address")
	flag.StringVar(&bindAddr, "bind-addr", ":8181", "input bind address")
	flag.Parse()
}

func main() {
	_, _, err := tracer.NewTracer(serviceName, traceAddr)
	if err != nil {
		panic(err.Error())
	}
	defer tracer.Close()

	initGrpcClient()

	server()
}

func initGrpcClient() {
	conn, err := grpc.Dial("127.0.0.1:3001", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("connect server error: %v", err)
	}

	// init grpc client
	gclient = pb.NewUserServiceClient(conn)
}

func server() {
	r := gin.Default()
	r.Use(tracer.TracingMiddleware(serviceName))
	r.GET("/ping", handlePing)
	r.Run(bindAddr)
}

func handlePing(c *gin.Context) {
	ctx := c.Request.Context()

	handleLogin(ctx)
	hanldePost(ctx)
	handleCall(ctx, []string{"aa", "bb"})

	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func handleLogin(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetCallerCache())
	defer span.Finish()

	fakeSleep()
	return
}

func hanldePost(ctx context.Context) {
	span, cctx := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer span.Finish()

	handleGrpcService(cctx, 1111)
	handleGrpcService(cctx, 2222)
}

func handleGrpcService(ctx context.Context, uid int) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer span.Finish()

	// inject header, fill biz header
	md := metadata.Pairs("key-111", "val-111")
	cctx, md := tracer.InjectGrpcMD(span, md, ctx)
	md.Set("key-222", "val-222")

	// request
	var rheader metadata.MD
	resp, err := gclient.GetUserInfo(
		cctx,
		&pb.UserRequest{ID: int32(uid)},
		grpc.Header(&rheader),
	)
	if err != nil {
		fmt.Println("grpc request failed, err: ", err.Error())
		return
	}

	span.SetTag("resp.body", resp.String())
	span.SetTag("req.md", md)
	span.SetTag("resp.header", rheader)
}

func handleCall(ctx context.Context, args interface{}) (reply string) {
	span, cctx := tracer.StartSpanFromContext(ctx, tracer.GetCallerCache())
	defer func() {
		span.SetTag("request", args)
		span.SetTag("reply", map[string]interface{}{"url": "xiaorui.cc", "age": 18, "addr": "china bj"})
		span.Finish()
	}()

	fakeSleep()

	handleInsideFunc(cctx, []string{"in1", "in2"})
	return time.Now().String()
}

func handleInsideFunc(ctx context.Context, params interface{}) (reply string) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetCallerCache())
	defer func() {
		span.SetTag("request", params)
		span.SetTag("reply", reply)
		span.Finish()
	}()

	fakeSleep()

	handleRedis(ctx)
	handleDatabase(ctx)

	return time.Now().String()
}

func handleRedis(ctx context.Context) {
	getUserFromRedis(ctx)
	updateDataFromRedis(ctx)
}

func getUserFromRedis(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, "get_user_from_redis")
	defer span.Finish()

	span.SetTag("command", "hgetall user:1122")
	fakeSleepN(500)
	return
}

func updateDataFromRedis(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, "update_data_from_redis")
	defer span.Finish()

	span.SetTag("command", "update data")
	fakeSleepN(500)
	return
}

func handleDatabase(ctx context.Context) {
	getUserFromDB(ctx)
	updateDataFromDB(ctx)
}

func getUserFromDB(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, "get_user_from_db")
	defer span.Finish()

	span.SetTag("command", "select * from user")
	fakeSleepN(500)
	return
}

func updateDataFromDB(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, "update_data_from_db")
	defer span.Finish()

	span.SetTag("sql", "update user set key = value where uid = 123")
	fakeSleepN(500)
	return
}

func fakeSleep() {
	time.Sleep(time.Duration(rand.Intn(1000) * int(time.Millisecond)))
}

func fakeSleepN(n int) {
	time.Sleep(time.Duration(rand.Intn(n) * int(time.Millisecond)))
}
