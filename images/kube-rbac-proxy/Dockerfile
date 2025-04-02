# Copyright 2021 Kube RBAC Proxy Authors rights reserved.
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
#
# Modifications Copyright 2022 Cortex Labs, Inc.

ARG TARGETARCH, TARGETOS

FROM golang:1.20.4 AS builder
RUN git clone -b v0.14.1 --depth 1 https://github.com/brancz/kube-rbac-proxy /go/src/github.com/brancz/kube-rbac-proxy
WORKDIR /go/src/github.com/brancz/kube-rbac-proxy/cmd/kube-rbac-proxy
RUN go mod download
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build --installsuffix cgo -o kube-rbac-proxy .
RUN cp kube-rbac-proxy /usr/local/bin

FROM alpine:3.18
RUN apk add -U --no-cache ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /usr/local/bin/kube-rbac-proxy /kube-rbac-proxy
ENTRYPOINT ["/kube-rbac-proxy"]
EXPOSE 8080
