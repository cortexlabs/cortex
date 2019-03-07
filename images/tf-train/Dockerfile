FROM cortexlabs/tf-base

ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/lib/requirements.txt /src/lib/requirements.txt
RUN pip3 install -r /src/lib/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib
COPY pkg/workloads/tf_train /src/tf_train

ENTRYPOINT ["/usr/bin/python3", "/src/tf_train/train.py"]
