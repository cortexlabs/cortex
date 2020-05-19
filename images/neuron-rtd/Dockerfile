FROM amazonlinux:2

RUN echo $'[neuron] \n\
name=Neuron YUM Repository \n\
baseurl=https://yum.repos.neuron.amazonaws.com \n\
enabled=1' > /etc/yum.repos.d/neuron.repo

RUN rpm --import https://yum.repos.neuron.amazonaws.com/GPG-PUB-KEY-AMAZON-AWS-NEURON.PUB

RUN yum install -y aws-neuron-tools-1.0.6554.0
RUN yum install -y aws-neuron-runtime-1.0.6222.0
RUN yum install -y tar gzip procps-ng-3.3.10-26.amzn2.x86_64

ENV PATH="/opt/aws/neuron/bin:${PATH}"

RUN ln -s /sock/neuron.sock /run/neuron.sock

CMD neuron-rtd -g unix:/sock/neuron.sock -x
