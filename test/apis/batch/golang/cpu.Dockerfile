ARG TARGETARCH, TARGETOS

# Build the manager binary
FROM golang:1.15 as builder

# Copy the Go Modules manifests
COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY app/ /workspace/app

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -o main ./app

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/main .
USER nonroot:nonroot

ENTRYPOINT ["/main"]
