ARG TARGETARCH, TARGETOS

# Build the manager binary
FROM golang:1.17.3 as builder

# Copy the Go Modules manifests
COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY pkg/config pkg/config
COPY pkg/consts pkg/consts
COPY pkg/lib pkg/lib
COPY pkg/enqueuer pkg/enqueuer
COPY pkg/types pkg/types
COPY pkg/crds pkg/crds
COPY pkg/workloads pkg/workloads
COPY cmd/enqueuer cmd/enqueuer

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -o enqueuer ./cmd/enqueuer

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/enqueuer .
USER nonroot:nonroot

ENTRYPOINT ["/enqueuer"]
