ARG TARGETARCH, TARGETOS

FROM golang:1.17.3 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY pkg pkg
COPY cmd/proxy cmd/proxy
WORKDIR /workspace/cmd/proxy

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}  GO111MODULE=on go build -a -o /workspace/bin/proxy main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/bin/proxy .
USER 65532:65532

ENTRYPOINT ["/proxy"]
