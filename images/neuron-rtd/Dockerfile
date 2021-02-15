# Source: https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/docker-example/Dockerfile.neuron-rtd
FROM amazonlinux:2

RUN echo $'[neuron] \n\
name=Neuron YUM Repository \n\
baseurl=https://yum.repos.neuron.amazonaws.com \n\
enabled=1' > /etc/yum.repos.d/neuron.repo

RUN rpm --import https://yum.repos.neuron.amazonaws.com/GPG-PUB-KEY-AMAZON-AWS-NEURON.PUB

RUN yum install -y \
    aws-neuron-tools-1.4.2.0 \
    aws-neuron-runtime-1.4.3.0 \
    procps-ng-3.3.10-26.amzn2.x86_64 \
    gzip \
    tar \
    curl

ENV PATH="/opt/aws/neuron/bin:${PATH}"

RUN ln -s /sock/neuron.sock /run/neuron.sock

CMD neuron-rtd -g unix:/sock/neuron.sock --log-console
