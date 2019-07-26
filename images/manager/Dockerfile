FROM python:3.7-alpine3.10

WORKDIR /root

ENV PATH /root/.local/bin:$PATH

RUN pip3 install awscli --upgrade --user && \
    rm -rf /root/.cache/pip*

RUN apk add --no-cache bash curl gettext jq openssl pwgen

RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/0.2.1/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
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
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf linux-amd64 && \
    rm helm-v2.14.1-linux-amd64.tar.gz

RUN PW=$(pwgen -Bs1 12) && \
    WEBSITE=cortex.example.com && \
    openssl req \
       -subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=$WEBSITE" \
       -newkey rsa:2048 -nodes -keyout $WEBSITE.key \
       -x509 -days 3650 -out $WEBSITE.crt

RUN ISTIO_VERSION=1.2.2 && curl -L "https://github.com/istio/istio/releases/download/${ISTIO_VERSION}/istio-${ISTIO_VERSION}-linux.tar.gz" | tar xz && \
    ls && \
    mv ./istio-$ISTIO_VERSION/install/kubernetes/helm/istio ./manifests/ && \
    mv ./istio-$ISTIO_VERSION/install/kubernetes/helm/istio-init ./manifests/ && \
    rm -rf ./$ISTIO_VERSION

ENTRYPOINT ["/bin/bash"]
