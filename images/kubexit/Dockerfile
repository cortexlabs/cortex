# This file has been adapted from https://github.com/cortexlabs/kubexit
ARG TARGETARCH, TARGETOS

FROM golang:1.14 AS builder

RUN mkdir /tmp/kubexit
RUN git clone -b v0.1.0-cortex --depth 1  https://github.com/cortexlabs/kubexit.git /tmp/kubexit

WORKDIR /tmp/kubexit
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o kubexit ./cmd/kubexit

FROM alpine:3.11
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /tmp/kubexit/kubexit /bin/
ENTRYPOINT ["kubexit"]
