FROM golang:1.12.6 as builder

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    mv ./kubectl /tmp/kubectl

COPY go.mod go.sum /go/src/github.com/cortexlabs/cortex/
WORKDIR /go/src/github.com/cortexlabs/cortex
RUN go mod download

COPY pkg /go/src/github.com/cortexlabs/cortex/pkg
WORKDIR /go/src/github.com/cortexlabs/cortex/pkg/operator
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o operator .


FROM alpine:3.7

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /tmp/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

COPY pkg/transformers /src/transformers
COPY pkg/aggregators /src/aggregators
COPY pkg/estimators /src/estimators

COPY --from=builder /go/src/github.com/cortexlabs/cortex/pkg/operator/operator /root/
RUN chmod +x /root/operator

EXPOSE 8888
ENTRYPOINT ["/root/operator"]
