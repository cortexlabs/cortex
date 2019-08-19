FROM cortexlabs/spark

RUN pip3 install pytest mock

COPY pkg/workloads /src

COPY images/test/run.sh /src/run.sh

WORKDIR /src/cortex

ENTRYPOINT ["/bin/bash"]
CMD ["/src/run.sh"]
