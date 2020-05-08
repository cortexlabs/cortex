FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
    wget \
    gnupg && \
    echo "deb https://apt.repos.neuron.amazonaws.com bionic main" >> /etc/apt/sources.list.d/neuron.list && \
    wget -qO - https://apt.repos.neuron.amazonaws.com/GPG-PUB-KEY-AMAZON-AWS-NEURON.PUB | apt-key add - && \
    apt-get update -qq && apt-get install -y -q \
    aws-neuron-runtime \
    aws-neuron-tools \
    tensorflow-model-server-neuron && \
    apt-get clean -qq && rm -rf /var/lib/apt/lists/*
ENV PATH=/opt/aws/neuron/bin/:$PATH

COPY images/tf-serve-accelerator/run.sh /src/

CMD ["/src/run.sh"]