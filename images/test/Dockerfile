FROM python:3.6

COPY pkg/cortex/serve/cortex_internal.requirements.txt /src/cortex/serve/cortex_internal.requirements.txt

RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r /src/cortex/serve/cortex_internal.requirements.txt && \
    pip install pytest mock && \
    rm -rf /root/.cache/pip*

COPY pkg /src
COPY images/test/run.sh /src/run.sh

COPY pkg/cortex/serve/log_config.yaml /src/cortex/serve/log_config.yaml
ENV CORTEX_LOG_LEVEL DEBUG
ENV CORTEX_LOG_CONFIG_FILE /src/cortex/serve/log_config.yaml

RUN pip install --no-deps /src/cortex/serve/ && \
    rm -rf /root/.cache/pip*

WORKDIR /src/cortex/serve/cortex_internal/

ENTRYPOINT ["/bin/bash"]
CMD ["/src/run.sh"]
