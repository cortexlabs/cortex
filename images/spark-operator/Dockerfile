FROM golang:1.10.2-alpine as builder

ARG DEP_VERSION="0.4.1"
RUN apk add --no-cache bash git
ADD https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

RUN git clone https://github.com/GoogleCloudPlatform/spark-on-k8s-operator.git /spark-on-k8s-operator
RUN cd /spark-on-k8s-operator && git checkout 62db1d66dafa8f43d6974e17a2fa90e56f675312

WORKDIR ${GOPATH}/src/github.com/GoogleCloudPlatform/spark-on-k8s-operator
RUN cp -r /spark-on-k8s-operator/* ./
RUN dep ensure -v
RUN go generate && CGO_ENABLED=0 GOOS=linux go build -o /usr/bin/spark-operator


FROM cortexlabs/spark-base
COPY --from=builder /usr/bin/spark-operator /usr/bin/
RUN apt-get update -qq && apt-get install -y openssl -q && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

COPY --from=builder /spark-on-k8s-operator/hack/gencerts.sh /usr/bin/
ENTRYPOINT ["/usr/bin/spark-operator"]
