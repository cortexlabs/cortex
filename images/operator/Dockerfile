FROM golang:1.15 as builder

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.20.2/bin/linux/amd64/kubectl && \
    mv ./kubectl /tmp/kubectl

COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY pkg/config /workspace/pkg/config
COPY pkg/consts /workspace/pkg/consts
COPY pkg/lib /workspace/pkg/lib
COPY pkg/operator /workspace/pkg/operator
COPY pkg/types /workspace/pkg/types
COPY pkg/crds /workspace/pkg/crds
COPY pkg/workloads /workspace/pkg/workloads
WORKDIR /workspace/pkg/operator
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o operator .


FROM alpine:3.12

COPY --from=builder /tmp/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /workspace/pkg/operator/operator /root/
RUN chmod +x /root/operator

EXPOSE 8888
ENTRYPOINT ["/root/operator"]
