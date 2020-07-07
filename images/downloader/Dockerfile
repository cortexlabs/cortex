FROM ubuntu:18.04

ENV PYTHONPATH="${PYTHONPATH}:/src"

RUN apt-get update -qq && apt-get install -y -q \
        curl \
        python3.6 \
        python3.6-distutils \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
    python3.6 get-pip.py && \
    pip install --upgrade pip && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/downloader/requirements.txt /src/cortex/downloader/requirements.txt
RUN pip install -r /src/cortex/downloader/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/consts.py /src/cortex/
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/downloader /src/cortex/downloader

ENTRYPOINT ["/usr/bin/python3.6", "/src/cortex/downloader/download.py"]
