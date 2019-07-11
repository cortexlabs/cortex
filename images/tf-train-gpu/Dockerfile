FROM cortexlabs/tf-base-gpu

ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/cortex/lib/requirements.txt /src/cortex/lib/requirements.txt
RUN pip3 install -r /src/cortex/lib/requirements.txt && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/cortex/consts.py /src/cortex/
COPY pkg/workloads/cortex/lib /src/cortex/lib
COPY pkg/workloads/cortex/tf_train /src/cortex/tf_train

ENTRYPOINT ["/usr/bin/python3", "/src/cortex/tf_train/train.py"]
