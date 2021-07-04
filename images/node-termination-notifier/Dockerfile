FROM ubuntu:18.04

RUN apt update && \
    apt install libzmq3-dev python3-pip --no-install-recommends -y && \
    apt-get clean -qq && rm -rf /var/lib/apt/lists/*

ENV LC_ALL=C.UTF-8
ENV LANG=C.UTF-8

WORKDIR /app
COPY python/node-termination-notifier/requirements.txt .
RUN pip3 install -r requirements.txt

COPY python/node-termination-notifier/ .

ENTRYPOINT ["python3", "main.py"]
