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

COPY pkg/consts pkg/consts
COPY pkg/lib pkg/lib
COPY pkg/async-gateway pkg/async-gateway
COPY pkg/types pkg/types
COPY cmd/async-gateway cmd/async-gateway

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -installsuffix cgo -o async-gateway ./cmd/async-gateway

FROM alpine:3.18

RUN apk update && apk add ca-certificates

COPY --from=builder /workspace/async-gateway /root/
RUN chmod +x /root/async-gateway

ENTRYPOINT ["/root/async-gateway"]
