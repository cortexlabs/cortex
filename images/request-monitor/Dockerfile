FROM golang:1.15 as builder

WORKDIR /workspace
COPY go.mod go.sum /workspace/
RUN go mod download

COPY cmd/request-monitor /workspace
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o request-monitor .


FROM alpine:3.12

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /workspace/request-monitor /root/
RUN chmod +x /root/request-monitor

ENTRYPOINT ["/root/request-monitor"]
