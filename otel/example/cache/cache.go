package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net"
	"time"

	pb "github.com/rfyiamcool/grpc-example/simple/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rfyiamcool/go-tracer/otel"
)

var (
	bindAddr    = "0.0.0.0:3001"
	serviceName = "cache_server"
	url         string
)

func init() {
	flag.StringVar(&url, "url", "http://127.0.0.1:14268/api/traces", "input jaeger url")
	flag.Parse()
}

type userCache struct{}

func (s *userCache) GetUserInfo(ctx context.Context, req *pb.UserRequest) (resp *pb.UserResponse, err error) {
	reqHeader, _ := metadata.FromIncomingContext(ctx)
	log.Printf("recv request: %+v\n", req.ID)
	log.Printf("recv header: %+v\n", reqHeader)

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
	_, span := otel.Start(ctx, "post")
	defer span.End()

	fakeSleep()
}

func (s *userCache) handleClean(ctx context.Context) {
	_, span := otel.Start(ctx, "clean")
	defer span.End()

	fakeSleep()
}

func main() {
	tp, err := otel.New(serviceName, otel.WithMode(otel.ModeCollectorHttp), otel.WithAddress(url), otel.WithQueueSize(3000))
	if err != nil {
		log.Fatal(err)
	}
	defer tp.Shutdown(context.Background())

	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Println("server listen: ", bindAddr)

	grpcServer := grpc.NewServer(otel.GrpcUnaryServerOption())
	pb.RegisterUserServiceServer(grpcServer, &userCache{})
	grpcServer.Serve(listener)
}

func fakeSleep() {
	time.Sleep(time.Duration(rand.Intn(200) * int(time.Millisecond)))
}
