# This file has been adapted from https://github.com/brancz/kube-rbac-proxy
ARG TARGETARCH, TARGETOS

FROM golang:1.11 AS builder
RUN git clone -b v0.4.1 --depth 1 https://github.com/brancz/kube-rbac-proxy /go/src/github.com/brancz/kube-rbac-proxy
WORKDIR /go/src/github.com/brancz/kube-rbac-proxy
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build --installsuffix cgo -o kube-rbac-proxy github.com/brancz/kube-rbac-proxy \
    && cp kube-rbac-proxy /usr/local/bin

FROM alpine:3.8
RUN apk add -U --no-cache ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /usr/local/bin/kube-rbac-proxy .
ENTRYPOINT ["./kube-rbac-proxy"]
EXPOSE 8080
