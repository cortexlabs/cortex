FROM cortexlabs/spark-base

RUN apt-get update -qq && apt-get install -y -q \
        python3 \
        python3-dev \
        python3-pip \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    pip3 install --upgrade \
        pip \
        setuptools \
    && rm -rf /root/.cache/pip*

ENV PYSPARK_PYTHON=/usr/bin/python3
ENV PYTHONPATH="${SPARK_HOME}/python/lib/pyspark.zip:${SPARK_HOME}/python/lib/py4j-0.10.7-src.zip:${PYTHONPATH}"
ENV LIBRARY_PATH="/lib:/usr/lib"

# Pillow deps
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
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN cp $SPARK_HOME/kubernetes/dockerfiles/spark/entrypoint.sh /opt/
# Hide entrypoint.sh commands
RUN sed -i "/^set -ex$/c\set -e" /opt/entrypoint.sh

# Our code
ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/lib/requirements.txt /src/lib/requirements.txt
RUN pip3 install -r /src/lib/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib
COPY pkg/workloads/spark_job /src/spark_job

# $SPARK_HOME/conf gets clobbered by a volume that spark-on-k8s mounts (KubernetesClientApplication.scala)
RUN cp -r $SPARK_HOME/conf $SPARK_HOME/conf-custom
# This doesn't always hold (gets clobbered somewhere), so is reset in run.sh
ENV SPARK_CONF_DIR="${SPARK_HOME}/conf-custom"

COPY images/spark/run.sh /src/run.sh
RUN chmod +x /src/run.sh
ENTRYPOINT [ "/src/run.sh" ]
