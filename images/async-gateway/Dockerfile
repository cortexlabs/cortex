FROM golang:1.15 as builder

COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY pkg/consts /workspace/pkg/consts
COPY pkg/lib /workspace/pkg/lib
COPY pkg/async-gateway /workspace/pkg/async-gateway
COPY pkg/types /workspace/pkg/types
WORKDIR /workspace/pkg/async-gateway
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o async-gateway .

FROM alpine:3.12

RUN apk update && apk add ca-certificates

COPY --from=builder /workspace/pkg/async-gateway/async-gateway /root/
RUN chmod +x /root/async-gateway

ENTRYPOINT ["/root/async-gateway"]
