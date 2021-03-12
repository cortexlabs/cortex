FROM golang:1.15 as builder

COPY async-gateway/go.mod async-gateway/go.sum /workspace/async-gateway/
WORKDIR /workspace/async-gateway/
RUN go mod download

COPY async-gateway/*.go /workspace/async-gateway/
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o async-gateway .

FROM alpine:3.12

RUN apk update && apk add ca-certificates

COPY --from=builder /workspace/async-gateway/async-gateway /root/

ENTRYPOINT ["/root/async-gateway"]
