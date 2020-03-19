FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
        build-essential \
        curl \
        libfreetype6-dev \
        libpng-dev \
        libzmq3-dev \
        pkg-config \
        rsync \
        software-properties-common \
        unzip \
        zlib1g-dev \
        git \
        libsm6 \
        libxext6 \
        libxrender-dev \
        libsndfile1 \
        locales \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8 LANGUAGE=en_US.en LC_ALL=en_US.UTF-8

ENV PYTHONPATH="${PYTHONPATH}:/src:/mnt/project"
ENV PATH=/opt/conda/bin:$PATH

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
COPY pkg/workloads/cortex/serve/onnx-cpu.requirements.txt /src/cortex/serve/requirements.txt
COPY pkg/workloads/cortex/environment.yaml /src/cortex/environment.yaml

RUN curl https://repo.anaconda.com/miniconda/Miniconda3-4.7.12.1-Linux-x86_64.sh --output ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p /opt/conda && \
    /opt/conda/bin/conda install --force-reinstall -c conda-forge python=3.6.9 && \
    /opt/conda/bin/conda env create -n env -f /src/cortex/environment.yaml && \
    /opt/conda/bin/conda clean -a && \
    rm -rf ~/.cache ~/miniconda.sh && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" > ~/.env && \
    echo "conda activate env" >> ~/.env && \
    echo "source ~/.env" >> ~/.bashrc

ENV BASH_ENV=~/.env

COPY pkg/workloads/cortex/consts.py /src/cortex
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/serve /src/cortex/serve

ENTRYPOINT ["/src/cortex/serve/run.sh"]
