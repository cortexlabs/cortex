FROM python:3.7-alpine3.10

WORKDIR /root

ENV PATH /root/.local/bin:$PATH

RUN pip3 install awscli --upgrade --user && \
    rm -rf /root/.cache/pip*

RUN apk add --no-cache bash curl gettext jq git openssl pwgen

RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/0.1.40/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/eksctl /usr/local/bin

RUN curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.13.7/2019-06-11/bin/linux/amd64/aws-iam-authenticator && \
    chmod +x ./aws-iam-authenticator && \
    mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.15.0/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

COPY manager /root

RUN curl -LO https://get.helm.sh/helm-v2.14.1-linux-amd64.tar.gz && \
    tar -zxvf helm-v2.14.1-linux-amd64.tar.gz && \
    chmod +x linux-amd64/helm && \
    mv linux-amd64/helm /usr/local/bin/helm

RUN git clone https://github.com/nicholasjackson/mtls-go-example && \
    cd /root/mtls-go-example && \
    PW=$(pwgen -Bs1 12) && yes | ./generate.sh cortex.example.com $PW && \
    mkdir ../cortex.example.com && mv 1_root 2_intermediate 3_application 4_client ../cortex.example.com

RUN curl -L https://git.io/getLatestIstio | ISTIO_VERSION=1.2.2 sh - && \
    mv ./istio-1.2.2/install/kubernetes/helm/istio ./manifests/ && \
    mv ./istio-1.2.2/install/kubernetes/helm/istio-init ./manifests/

ENTRYPOINT ["/bin/bash"]
