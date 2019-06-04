FROM cortexlabs/tf-base-gpu

ARG TF_SERV_VERSION="1.13.0"

RUN curl -o tensorflow-model-server.deb http://storage.googleapis.com/tensorflow-serving-apt/pool/tensorflow-model-server-${TF_SERV_VERSION}/t/tensorflow-model-server/tensorflow-model-server_${TF_SERV_VERSION}_all.deb
RUN dpkg -i tensorflow-model-server.deb

ENTRYPOINT ["tensorflow_model_server"]
