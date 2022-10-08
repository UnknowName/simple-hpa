FROM golang as builder

ENV GOPROXY="https://goproxy.io,direct"

ADD ./  /simple-hpa

RUN cd /simple-hpa \
    && go build src/simple-hpa.go \
    && chmod +x simple-hpa

FROM debian

ENV TZ=Asia/Shanghai

COPY --from=builder /simple-hpa/simple-hpa /simple-hpa
COPY --from=builder /simple-hpa/config.yaml /config.yaml

EXPOSE 514/udp

ENTRYPOINT /simple-hpa