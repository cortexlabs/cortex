FROM golang:1.14.7 as builder

COPY images/request-monitor/go.mod images/request-monitor/go.sum /go/src/github.com/cortexlabs/cortex/images/request-monitor/
WORKDIR /go/src/github.com/cortexlabs/cortex/images/request-monitor
RUN go mod download

COPY images/request-monitor/request-monitor.go /go/src/github.com/cortexlabs/cortex/images/request-monitor/
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o request-monitor .


FROM alpine:3.12

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /go/src/github.com/cortexlabs/cortex/images/request-monitor/request-monitor /root/
RUN chmod +x /root/request-monitor

ENTRYPOINT ["/root/request-monitor"]
