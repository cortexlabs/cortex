# Source: https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/docker-example/Dockerfile.tf-serving
FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
    gettext-base \
    supervisor \
    curl \
    wget \
    netcat \
    gnupg && \
    echo "deb https://apt.repos.neuron.amazonaws.com bionic main" >> /etc/apt/sources.list.d/neuron.list && \
    wget -qO - https://apt.repos.neuron.amazonaws.com/GPG-PUB-KEY-AMAZON-AWS-NEURON.PUB | apt-key add - && \
    apt-get update -qq && apt-get install -y -q \
    aws-neuron-tools=1.4.2.0 \
    aws-neuron-runtime=1.4.3.0 \
    tensorflow-model-server-neuron=1.15.0.1.2.2.0 && \
    apt-get clean -qq && rm -rf /var/lib/apt/lists/*

ENV PATH=/opt/aws/neuron/bin/:$PATH

COPY images/tensorflow-serving-inf/run.sh /src/
COPY images/tensorflow-serving-inf/supervisord.conf /tmp/supervisord.conf
COPY images/tensorflow-serving-inf/template.conf /tmp/template.conf
RUN chmod +x /src/run.sh

ENTRYPOINT ["/src/run.sh"]
