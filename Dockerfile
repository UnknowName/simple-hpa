FROM golang as builder

ENV GOPROXY="https://goproxy.io,direct"

ADD ./  /simple-hpa

RUN cd /simple-hpa \
    && go build src/auto-scale.go \
    && chmod +x auto-scale

FROM debian

ENV TZ=Asia/Shanghai

COPY --from=builder /simple-hpa/auto-scale /auto-scale
COPY --from=builder /simple-hpa/config.yaml /config.yaml

EXPOSE 514/udp

ENTRYPOINT /auto-scale