FROM ubuntu:18.04

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

ENV PATH=/opt/conda/bin:$PATH
ENV PYTHONVERSION=3.6.9

# conda needs an untainted base environment to function properly
# that's why a new separate conda environment is created
RUN curl "https://repo.anaconda.com/miniconda/Miniconda3-4.7.12.1-Linux-x86_64.sh" --output ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p /opt/conda && \
    rm -rf ~/.cache ~/miniconda.sh

# split the conda installations because the dev boxes have limited memory
RUN /opt/conda/bin/conda create -n env -c conda-forge python=$PYTHONVERSION pip=19.* && \
    /opt/conda/bin/conda clean -a && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" > ~/.env && \
    echo "conda activate env" >> ~/.env && \
    echo "source ~/.env" >> ~/.bashrc

ENV BASH_ENV=~/.env
SHELL ["/bin/bash", "-c"]

COPY pkg/cortex/serve/serve.requirements.txt /src/cortex/serve/serve.requirements.txt
COPY pkg/cortex/serve/onnx-cpu.requirements.txt /src/cortex/serve/image.requirements.txt
COPY pkg/cortex/serve/cortex_internal.requirements.txt /src/cortex/serve/cortex_internal.requirements.txt
RUN pip install --no-cache-dir \
    -r /src/cortex/serve/serve.requirements.txt \
    -r /src/cortex/serve/image.requirements.txt \
    -r /src/cortex/serve/cortex_internal.requirements.txt


ARG SLIM="false"
RUN test "${SLIM}" = "true" || ( \
        # libsm6, libxext6, and libxrender-dev are required for opencv
        # libpng-dev is required for some image processing libaries
        # libsndfile1 is required for some audio processing libaries
        # libgl1-mesa-glx to libxtst6 are for qt (for GUI packages): https://docs.anaconda.com/anaconda/install/linux/
        apt-get update -qq && apt-get install -y -q \
            libsm6 \
            libxext6 \
            libxrender-dev \
            libpng-dev \
            libsndfile1 \
            libgl1-mesa-glx \
            libegl1-mesa \
            libxrandr2 \
            libxss1 \
            libxcursor1 \
            libxcomposite1 \
            libasound2 \
            libxi6 \
            libxtst6 \
        && apt-get clean -qq && rm -rf /var/lib/apt/lists/* \
    )

COPY pkg/cortex/serve/ /src/cortex/serve
ENV CORTEX_LOG_CONFIG_FILE /src/cortex/serve/log_config.yaml

RUN pip install --no-deps /src/cortex/serve/
RUN mv /src/cortex/serve/init/bootloader.sh /etc/cont-init.d/bootloader.sh

ENTRYPOINT ["/init"]
