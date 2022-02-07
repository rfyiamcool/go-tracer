## tracer

simple trace sdk for OpenTracing and OpenTelemetry.

### Feature:

support component:

- simple api
- http middleware
- grpc middleware
- redis hook
- function span

### OpenTelemetry Usage:

#### init tracer

```go
import (
	"github.com/rfyiamcool/go-tracer/otel"
)

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
}
```

#### start span

```go
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
```

### OpenTracing Usage:

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

1. go run example/gateway/gw.go --trace-addr=127.0.0.1:6381
2. go run example/service/service.go --trace-addr=127.0.0.1:6381
3. go run example/cache/cache.go --trace-addr=127.0.0.1:6381

#### send request to gateway

Three ways to start the client !!!

- go run example/client/client.go --trace-addr=127.0.0.1:6381
- bash example/curl.sh, http request with custom x-trace-id.
- curl -vvv 127.0.0.1:8080/ping

```sh
$ > bash example/curl.sh

x-trace-id ===>  230b497c161e83e9:230b497c161e83e9:0000000000000000:1

*   Trying 127.0.0.1:8080...
* Connected to 127.0.0.1 (127.0.0.1) port 8080 (#0)
> GET /ping HTTP/1.1
> Host: 127.0.0.1:8080
> User-Agent: curl/7.71.1
> Accept: */*
> x-trace-id: 230b497c161e83e9:230b497c161e83e9:0000000000000000:1

* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Content-Type: application/json; charset=utf-8
< Span-Id: 07f59dea1f92f1dd
< Trace-Id: 230b497c161e83e9
< Date: Sun, 06 Feb 2022 01:29:55 GMT
< Content-Length: 18
<
* Connection #0 to host 127.0.0.1 left intact

{"message":"pong"}%
```

#### query trace in UI

1. open `http://${jaeger_query_addr}:16686/search` in chrome browser.
2. input trace-id in jaeger ui.

![](docs/jaeger_get_trace.jpg)