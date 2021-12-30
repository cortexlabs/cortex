# to replace when building the dockerfile
FROM ubuntu:18.04
ENV CORTEX_MODEL_SERVER_VERSION=0.2.0

RUN apt-get update -qq && apt-get install -y -q \
        build-essential \
        pkg-config \
        software-properties-common \
        curl \
        git \
        unzip \
        zlib1g-dev \
        locales \
        nginx=1.14.* \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN cd /tmp/ && \
    curl -L --output s6-overlay-amd64-installer "https://github.com/just-containers/s6-overlay/releases/download/v2.1.0.2/s6-overlay-amd64-installer" && \
    cd - && \
    chmod +x /tmp/s6-overlay-amd64-installer && /tmp/s6-overlay-amd64-installer / && rm /tmp/s6-overlay-amd64-installer

ENV S6_BEHAVIOUR_IF_STAGE2_FAILS 2

RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8 LANGUAGE=en_US.UTF-8 LC_ALL=en_US.UTF-8
ENV PATH=/opt/conda/bin:$PATH \
    PYTHONVERSION=3.6.9 \
    CORTEX_IMAGE_TYPE=python-handler-cpu \
    CORTEX_TF_SERVING_HOST=

# conda needs an untainted base environment to function properly
# that's why a new separate conda environment is created
RUN curl "https://repo.anaconda.com/miniconda/Miniconda3-4.7.12.1-Linux-x86_64.sh" --output ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p /opt/conda && \
    rm -rf ~/.cache ~/miniconda.sh

# split the conda installations to reduce memory usage when building the image
RUN /opt/conda/bin/conda create -n env -c conda-forge python=$PYTHONVERSION pip=19.* && \
    /opt/conda/bin/conda clean -a && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" > ~/.env && \
    echo "conda activate env" >> ~/.env && \
    echo "source ~/.env" >> ~/.bashrc

ENV BASH_ENV=~/.env
SHELL ["/bin/bash", "-c"]
RUN git clone --depth 1 -b ${CORTEX_MODEL_SERVER_VERSION} https://github.com/cortexlabs/nucleus && \
    cp -r nucleus/src/ /src/ && \
    rm -r nucleus/

RUN pip install --no-cache-dir \
    -r /src/cortex/serve.requirements.txt \
    -r /src/cortex/cortex_internal.requirements.txt
ENV CORTEX_LOG_CONFIG_FILE=/src/cortex/log_config.yaml \
    CORTEX_TELEMETRY_SENTRY_DSN="https://c334df915c014ffa93f2076769e5b334@sentry.io/1848098"

RUN mkdir -p /usr/local/cortex/ && \
    cp /src/cortex/init/install-core-dependencies.sh /usr/local/cortex/install-core-dependencies.sh && \
    chmod +x /usr/local/cortex/install-core-dependencies.sh && \
    /usr/local/cortex/install-core-dependencies.sh

RUN pip install --no-deps /src/cortex/ && \
    mv /src/cortex/init/bootloader.sh /etc/cont-init.d/bootloader.sh

COPY ./requirements.txt /src/project/
COPY ./model-server-config.yaml /src/project/model-server-config.yaml

ENV CORTEX_MODEL_SERVER_CONFIG /src/project/model-server-config.yaml
RUN /opt/conda/envs/env/bin/python /src/cortex/init/expand_server_config.py /src/project/model-server-config.yaml > /tmp/model-server-config.yaml && \
   eval $(/opt/conda/envs/env/bin/python /src/cortex/init/export_env_vars.py /tmp/model-server-config.yaml) && \
    if [ -f "/src/project/${CORTEX_DEPENDENCIES_PIP}" ]; then pip --no-cache-dir install -r "/src/project/${CORTEX_DEPENDENCIES_PIP}"; fi && \
    /usr/local/cortex/install-core-dependencies.sh

COPY ./ /src/project/
RUN mv /tmp/model-server-config.yaml /src/project/model-server-config.yaml

ENTRYPOINT ["/init"]
