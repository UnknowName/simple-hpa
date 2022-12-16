FROM golang as builder

ENV GOPROXY="https://goproxy.io,direct"

ADD ./  /simple-hpa

RUN cd /simple-hpa \
    && go build src/auto-scale.go \
    && chmod +x auto-scale

FROM debian

ENV TZ=Asia/Shanghai

RUN apt-get update \
    && apt-get -qq install -y --no-install-recommends ca-certificates curl

COPY --from=builder /simple-hpa/auto-scale /auto-scale
COPY --from=builder /simple-hpa/config.yaml /config.yaml

EXPOSE 514/udp

ENTRYPOINT /auto-scale