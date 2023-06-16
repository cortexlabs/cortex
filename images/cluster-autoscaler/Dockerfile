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

FROM golang:1.19.7 AS builder
RUN git clone -b cluster-autoscaler-1.26.3-cortex --depth 1 https://github.com/PEAT-AI/autoscaler /k8s.io/autoscaler
WORKDIR /k8s.io/autoscaler/cluster-autoscaler
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build --installsuffix cgo -o cluster-autoscaler k8s.io/autoscaler/cluster-autoscaler \
    && cp cluster-autoscaler /usr/local/bin

FROM alpine:3.18
RUN apk add -U --no-cache ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /usr/local/bin/cluster-autoscaler .
