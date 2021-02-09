FROM python:3.6-alpine3.12

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

RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/0.36.2/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/eksctl /usr/local/bin

RUN curl -o aws-iam-authenticator https://amazon-eks.s3.us-west-2.amazonaws.com/1.17.9/2020-08-04/bin/linux/amd64/aws-iam-authenticator && \
    chmod +x ./aws-iam-authenticator && \
    mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.19.0/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

ENV ISTIO_VERSION 1.7.3
RUN curl -L https://istio.io/downloadIstio | sh -

RUN curl https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-317.0.0-linux-x86_64.tar.gz > /tmp/google-cloud-sdk.tar.gz && \
    mkdir -p /usr/local/gcloud && \
    tar -C /usr/local/gcloud -xvf /tmp/google-cloud-sdk.tar.gz && \
    /usr/local/gcloud/google-cloud-sdk/install.sh && \
    rm /tmp/google-cloud-sdk.tar.gz
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin


RUN curl https://get.helm.sh/helm-v3.5.2-linux-amd64.tar.gz > /tmp/helm.tar.gz && \
    mkdir -p /usr/local/bin/helm && \
    tar -C /tmp/ -xvzf /tmp/helm.tar.gz && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    rm -r /tmp/linux-amd64 && \
    rm /tmp/helm.tar.gz

COPY manager /root/manager/
COPY charts /root/charts/

ENTRYPOINT ["/bin/bash"]
