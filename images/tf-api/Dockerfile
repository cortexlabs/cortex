FROM cortexlabs/tf-base

ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/lib/requirements.txt /src/lib/requirements.txt
COPY pkg/workloads/tf_api/requirements.txt /src/tf_api/requirements.txt
RUN pip3 install -r /src/lib/requirements.txt && \
    pip3 install -r /src/tf_api/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib
COPY pkg/workloads/tf_api /src/tf_api

ENTRYPOINT ["/usr/bin/python3", "/src/tf_api/api.py"]
