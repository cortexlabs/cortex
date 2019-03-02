FROM cortexlabs/tf-base

ENV PYTHONPATH="/src:${PYTHONPATH}"

COPY pkg/workloads/lib/requirements* /src/lib/
COPY pkg/workloads/tf_train/requirements* /src/tf_train/
RUN find /src -name "requirements*" | xargs printf -- '-r %s ' | xargs pip3 install && \
    rm -rf /root/.cache/pip*

COPY pkg/workloads/consts.py /src/
COPY pkg/workloads/lib /src/lib
COPY pkg/workloads/tf_train /src/tf_train

ENTRYPOINT ["/usr/bin/python3", "/src/tf_train/train.py"]
