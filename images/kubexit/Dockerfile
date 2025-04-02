# Copyright 2020 Karl Isenberg (https://github.com/karlkfi/kubexit)
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

RUN mkdir /tmp/kubexit
RUN git clone -b v0.1.0-cortex --depth 1  https://github.com/cortexlabs/kubexit.git /tmp/kubexit

WORKDIR /tmp/kubexit
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o kubexit ./cmd/kubexit

FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /tmp/kubexit/kubexit /bin/
ENTRYPOINT ["kubexit"]
