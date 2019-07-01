FROM python:3.7-alpine3.10

WORKDIR /root

ENV PATH /root/.local/bin:$PATH

RUN pip3 install awscli --upgrade --user

RUN apk add --no-cache bash curl gettext

RUN curl --location "https://github.com/weaveworks/eksctl/releases/download/0.1.38/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/eksctl /usr/local/bin

RUN curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.13.7/2019-06-11/bin/linux/amd64/aws-iam-authenticator && \
    chmod +x ./aws-iam-authenticator && \
    mv ./aws-iam-authenticator /usr/local/bin/aws-iam-authenticator

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.15.0/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

COPY manager /root

ENTRYPOINT ["/bin/bash"]
