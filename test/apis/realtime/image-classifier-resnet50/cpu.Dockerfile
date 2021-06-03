FROM tensorflow/serving:2.3.0

RUN apt-get update -qq && apt-get install -y --no-install-recommends -q \
        wget \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN TFS_PROBE_VERSION=1.0.1 \
    && wget -qO /bin/tfs_model_status_probe https://github.com/codycollier/tfs-model-status-probe/releases/download/v${TFS_PROBE_VERSION}/tfs_model_status_probe_${TFS_PROBE_VERSION}_linux_amd64 \
    && chmod +x /bin/tfs_model_status_probe

RUN mkdir -p /model/resnet50/ \
    && wget -qO- http://download.tensorflow.org/models/official/20181001_resnet/savedmodels/resnet_v2_fp32_savedmodel_NHWC_jpg.tar.gz | \
    tar --strip-components=2 -C /model/resnet50 -xvz

ENV CORTEX_PORT 8501

ENTRYPOINT tensorflow_model_server --rest_api_port=$CORTEX_PORT --rest_api_num_threads=8 --model_name="resnet50" --model_base_path="/model/resnet50"
