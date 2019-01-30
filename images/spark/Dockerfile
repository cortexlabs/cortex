# Copyright 2019 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


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
COPY pkg/workloads/spark_job/requirements.txt /src/spark_job/requirements.txt
RUN pip3 install -r /src/lib/requirements.txt && \
    pip3 install -r /src/spark_job/requirements.txt && \
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
