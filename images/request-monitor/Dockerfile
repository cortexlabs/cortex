FROM golang:1.13.5

RUN mkdir -p /go/src/github.com/cortexlabs/cortex

RUN apt-get update -qq && apt-get install -y -q iproute2

RUN go get github.com/aws/aws-sdk-go

COPY images/request-monitor/request-monitor.go /go/src/github.com/cortexlabs/cortex/
COPY images/request-monitor/run.sh /

ENTRYPOINT ["/run.sh"]
