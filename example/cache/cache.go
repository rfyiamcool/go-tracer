package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	pb "github.com/rfyiamcool/grpc-example/simple/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rfyiamcool/go-tracer"
)

var (
	serviceName = "cache_server"

	bindAddr  string
	traceAddr string
)

func init() {
	flag.StringVar(&traceAddr, "trace-addr", "127.0.0.1:6831", "tracer agent address")
	flag.StringVar(&bindAddr, "bind-addr", "0.0.0.0:3001", "input bind address")
	flag.Parse()
}

type userCache struct{}

func (s *userCache) GetUserInfo(ctx context.Context, req *pb.UserRequest) (resp *pb.UserResponse, err error) {
	reqHeader, _ := metadata.FromIncomingContext(ctx)
	log.Printf("recv request: %+v\n", req.ID)
	log.Printf("recv header: %+v\n", reqHeader)

	// create and send header
	header := metadata.New(
		map[string]string{"random-num": strconv.Itoa(int(rand.Int63()))},
	)
	tracer.GrpcSendHeader(ctx, header)

	// handle
	s.handlePost(ctx)
	s.handleClean(ctx)

	// resp
	return &pb.UserResponse{
		Name: "haha",
		Age:  18,
	}, nil
}

func (s *userCache) handlePost(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer func() {
		span.Finish()
	}()

	fakeSleep()
}

func (s *userCache) handleClean(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer func() {
		span.Finish()
	}()

	fakeSleep()
}

func main() {
	_, _, err := tracer.NewTracer(serviceName, traceAddr)
	if err != nil {
		panic(err.Error())
	}
	defer tracer.Close()

	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Println("server listen: ", bindAddr)

	grpcServer := grpc.NewServer(tracer.GrpcServerOption())
	pb.RegisterUserServiceServer(grpcServer, &userCache{})
	grpcServer.Serve(listener)
}

func fakeSleep() {
	time.Sleep(time.Duration(rand.Intn(200) * int(time.Millisecond)))
}
