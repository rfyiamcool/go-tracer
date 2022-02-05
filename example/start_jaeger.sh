docker run -d \
    -p 5775:5775/udp \
    -p 16686:16686 \
    -p 6831:6831/udp \
    -p 6832:6832/udp \
    -p 5778:5778 \
    -p 14268:14268 jaegertracing/all-in-one:latest