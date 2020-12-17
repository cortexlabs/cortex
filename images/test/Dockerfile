FROM python:3.6

COPY pkg/workloads/cortex/internal/requirements.txt /src/cortex/internal/requirements.txt

RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r /src/cortex/internal/requirements.txt && \
    pip install pytest mock && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads /src
COPY images/test/run.sh /src/run.sh

RUN pip install --no-deps /src/cortex/internal/ && \
    rm -rf /root/.cache/pip*

WORKDIR /src/cortex/internal/cortex/

ENTRYPOINT ["/bin/bash"]
CMD ["/src/run.sh"]
