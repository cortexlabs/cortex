FROM python:3.6-alpine3.10

WORKDIR /root

ENV PATH /root/.local/bin:$PATH

COPY manager/requirements.txt /root/requirements.txt

RUN pip install --upgrade pip && \
    pip install awscli --upgrade --user && \
    pip install -r /root/requirements.txt && \
    rm -rf /root/.cache/pip*

RUN apk add --no-cache bash curl gettext jq openssl

RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/0.11.1/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/eksctl /usr/local/bin

RUN curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.14.6/2019-08-22/bin/linux/amd64/aws-iam-authenticator && \
    chmod +x ./aws-iam-authenticator && \
    mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.16.2/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

RUN curl -LO https://get.helm.sh/helm-v2.16.0-linux-amd64.tar.gz && \
    tar -zxvf helm-v2.16.0-linux-amd64.tar.gz && \
    chmod +x linux-amd64/helm && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf linux-amd64 && \
    rm helm-v2.16.0-linux-amd64.tar.gz

RUN ISTIO_VERSION=1.4.2 && \
    curl -L "https://github.com/istio/istio/releases/download/${ISTIO_VERSION}/istio-${ISTIO_VERSION}-linux.tar.gz" | tar xz && \
    mkdir ./istio-manifests && \
    mv ./istio-${ISTIO_VERSION}/install/kubernetes/helm/* ./istio-manifests/ && \
    rm -rf ./istio-${ISTIO_VERSION}

COPY manager /root

ENTRYPOINT ["/bin/bash"]
