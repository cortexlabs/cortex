FROM golang:1.15 as builder

COPY go.mod go.sum /go/src/github.com/cortexlabs/cortex/
WORKDIR /go/src/github.com/cortexlabs/cortex
RUN go mod download

COPY pkg/consts /go/src/github.com/cortexlabs/cortex/pkg/consts
COPY pkg/lib /go/src/github.com/cortexlabs/cortex/pkg/lib
COPY pkg/async-gateway /go/src/github.com/cortexlabs/cortex/pkg/async-gateway
COPY pkg/types /go/src/github.com/cortexlabs/cortex/pkg/types
WORKDIR /go/src/github.com/cortexlabs/cortex/pkg/async-gateway
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o async-gateway .

FROM alpine:3.12

RUN apk update && apk add ca-certificates

COPY --from=builder /go/src/github.com/cortexlabs/cortex/pkg/async-gateway/async-gateway /root/
RUN chmod +x /root/async-gateway

ENTRYPOINT ["/root/async-gateway"]
