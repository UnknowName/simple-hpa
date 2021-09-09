FROM golang as builder

ENV GOPROXY="https://goproxy.io,direct"

ADD ./  /simple-hpa

RUN cd /simple-hpa \
    && go build src/simple-hpa.go

FROM debian

COPY --from=builder /simple-hpa/simple-hpa /simple-hpa
COPY --from=builder /simple-hpa/config.yml /config.yml

EXPOSE 514/udp

ENTRYPOINT /simple-hpa /config.yml