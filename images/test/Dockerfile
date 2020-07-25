FROM python:3.6

ENV PYTHONPATH="${PYTHONPATH}:/src"

COPY pkg/workloads/cortex/serve/requirements.txt /src/cortex/serve/requirements.txt

RUN pip install --upgrade pip && \
    pip install -r /src/cortex/serve/requirements.txt && \
    pip install pytest mock && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads /src

COPY images/test/run.sh /src/run.sh

WORKDIR /src/cortex

ENTRYPOINT ["/bin/bash"]
CMD ["/src/run.sh"]
