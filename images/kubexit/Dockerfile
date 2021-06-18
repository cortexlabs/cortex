FROM golang:1.14.1-alpine3.11 AS builder
RUN apk update && apk upgrade && \
    apk add bash git openssh

RUN mkdir /tmp/kubexit
RUN git clone -b v0.1.0-cortex --depth 1  https://github.com/cortexlabs/kubexit.git /tmp/kubexit

WORKDIR /tmp/kubexit
RUN CGO_ENABLED=0 GOOS=linux go build -o kubexit ./cmd/kubexit

FROM alpine:3.11
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /tmp/kubexit/kubexit /bin/
ENTRYPOINT ["kubexit"]
