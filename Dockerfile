FROM golang as builder

ENV GOPROXY="https://goproxy.io,direct"

ADD ./  /simple-hpa

RUN cd /simple-hpa \
    && go build src/simple-hpa.go

FROM alpine

COPY --from=builder /simple-hpa/simple-hpa /simple-hpa
COPY --from=builder /simple-hpa/config.yaml /config.yaml

EXPOSE 514/udp

CMD ["/simple-hpa", "/config.yaml"]