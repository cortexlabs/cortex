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

FROM golang:1.20.4 as builder

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.26.5/bin/linux/amd64/kubectl && \
    mv ./kubectl /tmp/kubectl

COPY go.mod go.sum /workspace/
WORKDIR /workspace
RUN go mod download

COPY pkg/config pkg/config
COPY pkg/consts pkg/consts
COPY pkg/lib pkg/lib
COPY pkg/operator pkg/operator
COPY pkg/types pkg/types
COPY pkg/crds pkg/crds
COPY pkg/workloads pkg/workloads
COPY cmd/operator cmd/operator

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o operator ./cmd/operator


FROM alpine:3.18

COPY --from=builder /tmp/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /workspace/operator /root/
RUN chmod +x /root/operator

EXPOSE 8888
ENTRYPOINT ["/root/operator"]
