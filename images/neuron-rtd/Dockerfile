FROM amazonlinux:2

RUN echo $'[neuron] \n\
name=Neuron YUM Repository \n\
baseurl=https://yum.repos.neuron.amazonaws.com \n\
enabled=1' > /etc/yum.repos.d/neuron.repo

RUN rpm --import https://yum.repos.neuron.amazonaws.com/GPG-PUB-KEY-AMAZON-AWS-NEURON.PUB

RUN yum install -y aws-neuron-tools
RUN yum install -y aws-neuron-runtime
RUN yum install -y tar gzip

ENV PATH="/opt/aws/neuron/bin:${PATH}"

# to make neuron-cli accessible
RUN ln -s /sock/neuron.sock /run/neuron.sock

CMD neuron-rtd -g unix:/sock/neuron.sock