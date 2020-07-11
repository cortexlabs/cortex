FROM ubuntu:18.04

RUN apt-get update -qq && apt-get install -y -q \
    wget \
    gnupg && \
    echo "deb https://apt.repos.neuron.amazonaws.com bionic main" >> /etc/apt/sources.list.d/neuron.list && \
    wget -qO - https://apt.repos.neuron.amazonaws.com/GPG-PUB-KEY-AMAZON-AWS-NEURON.PUB | apt-key add - && \
    apt-get update -qq && apt-get install -y -q \
    aws-neuron-tools=1.0.6554.0 \
    aws-neuron-runtime=1.0.6222.0 && \
    apt-get clean -qq && rm -rf /var/lib/apt/lists/*
ENV PATH=/opt/aws/neuron/bin/:$PATH

RUN apt-get update -qq && apt-get install -y -q \
    build-essential \
    pkg-config \
    software-properties-common \
    curl \
    git \
    unzip \
    zlib1g-dev \
    locales \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8 LANGUAGE=en_US.en LC_ALL=en_US.UTF-8

ENV PYTHONPATH="${PYTHONPATH}:/src"
ENV PATH=/opt/conda/bin:$PATH
ENV PYTHONVERSION=3.6.9

# conda needs an untainted base environment to function properly
# that's why a new separate conda environment is created
RUN curl "https://repo.anaconda.com/miniconda/Miniconda3-4.7.12.1-Linux-x86_64.sh" --output ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p /opt/conda && \
    rm -rf ~/.cache ~/miniconda.sh

# split the conda installations because the dev boxes have limited memory
RUN /opt/conda/bin/conda create -n env -c conda-forge python=$PYTHONVERSION pip && \
    /opt/conda/bin/conda clean -a && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" > ~/.env && \
    echo "conda activate env" >> ~/.env && \
    echo "source ~/.env" >> ~/.bashrc

ENV BASH_ENV=~/.env
SHELL ["/bin/bash", "-c"]

COPY pkg/workloads/cortex/serve/requirements.txt /src/cortex/serve/requirements.txt
RUN pip install --no-cache-dir -r \
    /src/cortex/serve/requirements.txt

ARG SLIM="false"
RUN test "${SLIM}" = "true" || ( \
        # libsm6, libxext6, and libxrender-dev are required for opencv
        # libpng-dev is required for some image processing libaries
        # libsndfile1 is required for some audio processing libaries
        apt-get update -qq && apt-get install -y -q \
            libsm6 \
            libxext6 \
            libxrender-dev \
            libpng-dev \
            libsndfile1 \
        && apt-get clean -qq && rm -rf /var/lib/apt/lists/* \
        && pip install --no-cache-dir --extra-index-url https://pip.repos.neuron.amazonaws.com \
            neuron-cc==1.0.9410.0+6008239556 \
            tensorflow-neuron==1.15.0.1.0.1333.0 \
            torch-neuron==1.0.825.0 \
            cloudpickle==1.4.1 \
            Cython==0.29.17 \
            joblib==0.14.1 \
            nltk==3.5 \
            np-utils==0.5.12.1 \
            opencv-python==4.2.0.34 \
            pandas==1.0.3 \
            Pillow==6.2.2 \
            scikit-image==0.17.1 \
            scikit-learn==0.22.2.post1 \
            scipy==1.3.2 \
            six==1.14.0 \
            statsmodels==0.11.1 \
            sympy==1.5.1 \
            numpy==1.16.5 \
        && pip install --no-cache-dir --no-deps \
            torchvision==0.4.2 \
    )

COPY pkg/workloads/cortex/consts.py /src/cortex
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/serve /src/cortex/serve

ENTRYPOINT ["/src/cortex/serve/run.sh"]
