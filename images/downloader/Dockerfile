FROM ubuntu:18.04

ENV PYTHONPATH="/src:${PYTHONPATH}"

RUN apt-get update -qq && apt-get install -y -q \
        python3 \
        python3-dev \
        python3-pip \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    pip3 install --upgrade \
        pip \
        setuptools \
    && rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
RUN pip3 install -r /src/cortex/lib/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/consts.py /src/cortex/
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/downloader /src/cortex/downloader

ENTRYPOINT ["/usr/bin/python3", "/src/cortex/downloader/download.py"]
