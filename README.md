## tracer

easy tracer api for jaeger

### Feature:

support component:

- http middleware
- grpc middleware
- redis hook
- function middleware
- easy api

### Usage:

#### init tracer

```go
func NewTracer(serviceName string, addr string, fns ...optionFunc) (opentracing.Tracer, io.Closer, error) {
```

#### start span

```go
func handleLogic(ctx context.Context, params interface{}) error {
	span, cctx := tracer.StartSpanFromContext(ctx, tracer.GetFunc())
	defer func() {
		span.SetTag("request", params)
		span.Finish()
	}()

	resp, err := httpGet(cctx, "http://127.0.0.1:8181/ping", map[string]interface{}{"k1": "v1"})
	if err != nil {
		return err
	}

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
```

#### gin middleware

```go
r := gin.Default()
r.Use(tracer.TracingMiddleware(serviceName))
r.GET("/ping", handlePing)
r.Run(bindAddr)
```

#### grpc server interceptor

```go
listener, err := net.Listen("tcp", bindAddr)
grpcServer := grpc.NewServer(tracer.GrpcServerOption())
grpcServer.Serve(listener)
```

`For more usage, please see the code !!!`

### full example

[tracer code example](http://git.hualala.com/gopkg/tracer/example/)

#### request path:

![](docs/trace.jpg)

- client
- gateway
- service
- cache

#### start jaeger `all in one` container

```sh
bash example/start_jaeger.sh
```

#### start servers

1. go run example/gateway/gw.go
2. go run example/service/service.go
3. go run example/cache/cache.go

#### send request to gateway

Three ways to start the client !!!

- go run example/client/client.go
- bash example/curl.sh, http request with custom x-trace-id.
- curl -vvv 127.0.0.1:8080/ping

```sh
$ > bash example/curl.sh

x-trace-id ===>  73283a59e2a54dfd:73283a59e2a54dfd:0000000000000000:1

*   Trying 127.0.0.1:8080...
* Connected to 127.0.0.1 (127.0.0.1) port 8080 (#0)
> GET /ping HTTP/1.1
> Host: 127.0.0.1:8080
> User-Agent: curl/7.71.1
> Accept: */*
> x-trace-id: 73283a59e2a54dfd:73283a59e2a54dfd:0000000000000000:1


* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Content-Type: application/json; charset=utf-8
< Span-Id: 01d17b2f2779d185
< Trace-Id: 73283a59e2a54dfd
< Date: Wed, 02 Feb 2022 05:08:00 GMT
< Content-Length: 18
<
* Connection #0 to host 127.0.0.1 left intact
{"message":"pong"}%
```

#### query trace in UI

1. open `http://${jaeger_query_addr}:16686/search`
2. input trace-id in jaeger ui
