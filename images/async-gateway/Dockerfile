ARG TARGETARCH, TARGETOS

FROM golang:1.17.3 as builder

COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY pkg/consts pkg/consts
COPY pkg/lib pkg/lib
COPY pkg/async-gateway pkg/async-gateway
COPY pkg/types pkg/types
COPY cmd/async-gateway cmd/async-gateway

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -installsuffix cgo -o async-gateway ./cmd/async-gateway

FROM alpine:3.14

RUN apk update && apk add ca-certificates

COPY --from=builder /workspace/async-gateway /root/
RUN chmod +x /root/async-gateway

ENTRYPOINT ["/root/async-gateway"]
