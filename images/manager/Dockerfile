FROM python:3.6-alpine3.14

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

RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/v0.67.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/eksctl /usr/local/bin

RUN curl -o aws-iam-authenticator https://amazon-eks.s3.us-west-2.amazonaws.com/1.21.2/2021-07-05/bin/linux/amd64/aws-iam-authenticator && \
    chmod +x ./aws-iam-authenticator && \
    mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.22.3/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

RUN curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.1.2/kustomize_v4.1.2_linux_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/kustomize /usr/local/bin

ENV ISTIO_VERSION 1.11.4
RUN curl -L https://istio.io/downloadIstio | sh -

COPY manager /root
COPY pkg/crds/config /root/config

ENTRYPOINT ["/bin/bash"]
