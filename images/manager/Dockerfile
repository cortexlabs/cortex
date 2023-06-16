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

FROM python:3.7-alpine3.18

WORKDIR /root

ENV PATH /root/.local/bin:$PATH
ENV AWS_RETRY_MODE standard
ENV AWS_MAX_ATTEMPTS 10

COPY manager/requirements.txt /root/requirements.txt

RUN pip install --upgrade pip && \
    pip install awscli --upgrade --user && \
    pip install -r /root/requirements.txt && \
    rm -rf /root/.cache/pip*

RUN apk add --no-cache bash curl gettext jq openssl

RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/v0.143.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/eksctl /usr/local/bin

RUN curl -o aws-iam-authenticator curl -Lo aws-iam-authenticator https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v0.5.9/aws-iam-authenticator_0.5.9_linux_amd64 && \
    chmod +x ./aws-iam-authenticator && \
    mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.26.5/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

RUN curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.1.2/kustomize_v4.1.2_linux_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/kustomize /usr/local/bin

ENV ISTIO_VERSION 1.17.2
RUN curl -L https://istio.io/downloadIstio | sh -

COPY manager /root
COPY pkg/crds/config /root/config

ENTRYPOINT ["/bin/bash"]
