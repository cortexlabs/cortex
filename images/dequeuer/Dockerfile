# Copyright 2022 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG TARGETARCH, TARGETOS

FROM golang:1.20.4 as builder

COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY pkg/config pkg/config
COPY pkg/consts pkg/consts
COPY pkg/lib pkg/lib
COPY pkg/dequeuer pkg/dequeuer
COPY pkg/probe pkg/probe
COPY pkg/types pkg/types
COPY pkg/crds pkg/crds
COPY pkg/workloads pkg/workloads
COPY cmd/dequeuer cmd/dequeuer

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -o dequeuer ./cmd/dequeuer

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/dequeuer .
USER nonroot:nonroot

ENTRYPOINT ["/dequeuer"]
