FROM python:3.6

ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
RUN pip install -r /src/cortex/lib/requirements.txt && \
    pip install pytest mock && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads /src

COPY images/test/run.sh /src/run.sh

WORKDIR /src/cortex

ENTRYPOINT ["/bin/bash"]
CMD ["/src/run.sh"]
