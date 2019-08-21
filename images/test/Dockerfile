FROM python:3.7

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
RUN pip3 install -r /src/cortex/lib/requirements.txt && \
    pip3 install pytest mock && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads /src

COPY images/test/run.sh /src/run.sh

WORKDIR /src/cortex

ENTRYPOINT ["/bin/bash"]
CMD ["/src/run.sh"]
