FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
    gettext-base \
    supervisor \
    wget \
    netcat \
    gnupg && \
    echo "deb https://apt.repos.neuron.amazonaws.com bionic main" >> /etc/apt/sources.list.d/neuron.list && \
    wget -qO - https://apt.repos.neuron.amazonaws.com/GPG-PUB-KEY-AMAZON-AWS-NEURON.PUB | apt-key add - && \
    apt-get update -qq && apt-get install -y -q \
    aws-neuron-tools=1.0.6554.0 \
    aws-neuron-runtime=1.0.6222.0 \
    tensorflow-model-server-neuron && \
    apt-get clean -qq && rm -rf /var/lib/apt/lists/*

ENV PATH=/opt/aws/neuron/bin/:$PATH

RUN mkdir /etc/tfs && echo "model_config_list {}" > /etc/tfs/model_config_server.conf

COPY images/tensorflow-serving-inf/run.sh /src/
COPY images/tensorflow-serving-inf/supervisord.conf /tmp/supervisord.conf
COPY images/tensorflow-serving-inf/template.conf /tmp/template.conf
RUN chmod +x /src/run.sh

ENTRYPOINT ["/src/run.sh"]
