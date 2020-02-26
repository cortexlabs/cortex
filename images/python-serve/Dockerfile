FROM ubuntu:18.04
# FROM tiangolo/uvicorn-gunicorn:python3.6

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
        python3.6-dev \
        python3.6-distutils \
        git \
        libsm6 \
        libxext6 \
        libxrender-dev \
        locales \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
    python3.6 get-pip.py && \
    pip install --upgrade pip && \
    rm -rf /root/.cache/pip*

RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8 LANGUAGE=en_US.en LC_ALL=en_US.UTF-8

ENV PYTHONPATH "${PYTHONPATH}:/src:/mnt/project"

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
COPY pkg/workloads/cortex/serve/python-cpu.requirements.txt /src/cortex/serve/requirements.txt
RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r /src/cortex/lib/requirements.txt && \
    pip install --no-cache-dir -r /src/cortex/serve/requirements.txt && \
    rm -rf /root/.cache/pip*

RUN pip install --upgrade pip && \
    pip install --no-cache-dir uvicorn gunicorn fastapi && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/consts.py /src/cortex
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/serve /src/cortex/serve

COPY images/python-serve//start.sh /start.sh
RUN chmod +x /start.sh

COPY images/python-serve//gunicorn_conf.py /gunicorn_conf.py

COPY images/python-serve//start-reload.sh /start-reload.sh
RUN chmod +x /start-reload.sh

ENV PYTHONPATH=/app:/src

EXPOSE 80

# Run the start script, it will check for an /app/prestart.sh script (e.g. for migrations)
# And then will start Gunicorn with Uvicorn
CMD ["/start.sh"]

COPY images/python-serve/app /app
WORKDIR /app/

ENV PORT 8888

# ENTRYPOINT ["/src/cortex/serve/run.sh"]
