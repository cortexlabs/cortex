# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY ./enqueuer/go.mod enqueuer/go.mod
COPY ./enqueuer/go.sum enqueuer/go.sum

# Copy the Go modules manifests from the cortex package
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

WORKDIR /workspace/enqueuer
RUN go mod download

# Copy the go source for the enqueuer and the cortex package, as it is a local dependency
COPY ./enqueuer /workspace/enqueuer
COPY ./pkg /workspace/pkg

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o enqueuer .

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/enqueuer .
USER nonroot:nonroot

ENTRYPOINT ["/enqueuer"]
