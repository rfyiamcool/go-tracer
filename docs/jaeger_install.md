
## jaeger installation document

[more installation document](https://www.jaegertracing.io/docs/1.31/deployment/)

### simple installation

default memory storage

```sh
docker run -d \
    -p 5775:5775/udp \
    -p 16686:16686 \
    -p 6831:6831/udp \
    -p 6832:6832/udp \
    -p 5778:5778 \
    -p 14268:14268 jaegertracing/all-in-one:latest
```

elasticsearch storage

```sh
# elasticsearch
docker run -d --name=elasticsearch -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e "xpack.security.enabled=false" docker.elastic.co/elasticsearch/elasticsearch:7.12.1

# kibana
docker run -d --name=kibana --link=elasticsearch -p 5601:5601
docker.elastic.co/kibana/kibana:7.12.1

# all-in-one
docker run -d --name jaeger \
  --link=elasticsearch \
  -e SPAN_STORAGE_TYPE=elasticsearch \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -e ES_SERVER_URLS=http://elasticsearch:9200 \
  -e ES_TAGS_AS_FIELDS_ALL=true \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one
```

### detail installation

```sh
# bind ip
ip=127.0.0.1

# elasticsearch
docker run -d --name=elasticsearch -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e "xpack.security.enabled=false" -e ES_JAVA_OPTS="-Xms512m -Xmx512m"  docker.elastic.co/elasticsearch/elasticsearch:7.12.1

# jaeger-collector
docker run -d --name=jaeger-collector -p 9411:9411 -p 14250:14250 -p 14268:14268 -p 14269:14269 -e SPAN_STORAGE_TYPE=elasticsearch -e ES_SERVER_URLS=http://${ip}:9200 jaegertracing/jaeger-collector

# jaeger-agent
docker run -d --name=jaeger-agent -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778/tcp -p 5775:5775/udp -e REPORTER_GRPC_HOST_PORT=${ip}:14250 -e LOG_LEVEL=debug jaegertracing/jaeger-agent

# jaeger-query
docker run -d --name=jaeger-query  -p 16686:16686 -p 16687:16687 -e SPAN_STORAGE_TYPE=elasticsearch -e ES_SERVER_URLS=http://${ip}:9200 jaegertracing/jaeger-query

# get
docker container ls

# rm
docker container stop jaeger-collector jaeger-agent jaeger-query
docker container rm jaeger-collector jaeger-agent jaeger-query
```