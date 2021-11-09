ARG TARGETARCH, TARGETOS

FROM golang:1.15 AS builder
RUN git clone -b cluster-autoscaler-1.21.1-cortex --depth 1 https://github.com/cortexlabs/autoscaler /k8s.io/autoscaler
WORKDIR /k8s.io/autoscaler/cluster-autoscaler
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build --installsuffix cgo -o cluster-autoscaler k8s.io/autoscaler/cluster-autoscaler \
    && cp cluster-autoscaler /usr/local/bin

FROM alpine:3.8
RUN apk add -U --no-cache ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /usr/local/bin/cluster-autoscaler .
