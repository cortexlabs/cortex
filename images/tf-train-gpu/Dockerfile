FROM cortexlabs/tf-base-gpu

ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/lib/requirements.txt /src/lib/requirements.txt
COPY pkg/workloads/tf_train/requirements.txt /src/tf_train/requirements.txt
RUN pip3 install -r /src/lib/requirements.txt && \
    pip3 install -r /src/tf_train/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib
COPY pkg/workloads/tf_train /src/tf_train

ENTRYPOINT ["/usr/bin/python3", "/src/tf_train/train.py"]
