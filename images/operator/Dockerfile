FROM golang:1.17.3 as builder

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.22.3/bin/linux/amd64/kubectl && \
    mv ./kubectl /tmp/kubectl

COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY pkg/config pkg/config
COPY pkg/consts pkg/consts
COPY pkg/lib pkg/lib
COPY pkg/operator pkg/operator
COPY pkg/types pkg/types
COPY pkg/crds pkg/crds
COPY pkg/workloads pkg/workloads
COPY cmd/operator cmd/operator

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o operator ./cmd/operator


FROM alpine:3.14

COPY --from=builder /tmp/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /workspace/operator /root/
RUN chmod +x /root/operator

EXPOSE 8888
ENTRYPOINT ["/root/operator"]
