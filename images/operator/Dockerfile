FROM golang:1.13.5 as builder

COPY go.mod go.sum /go/src/github.com/cortexlabs/cortex/
WORKDIR /go/src/github.com/cortexlabs/cortex
RUN go mod download

COPY pkg/consts /go/src/github.com/cortexlabs/cortex/pkg/consts
COPY pkg/lib /go/src/github.com/cortexlabs/cortex/pkg/lib
COPY pkg/operator /go/src/github.com/cortexlabs/cortex/pkg/operator
COPY pkg/types /go/src/github.com/cortexlabs/cortex/pkg/types
WORKDIR /go/src/github.com/cortexlabs/cortex/pkg/operator
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o operator .


FROM alpine:3.11

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /go/src/github.com/cortexlabs/cortex/pkg/operator/operator /root/
RUN chmod +x /root/operator

EXPOSE 8888
ENTRYPOINT ["/root/operator"]
