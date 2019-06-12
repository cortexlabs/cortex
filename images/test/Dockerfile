FROM cortexlabs/spark

RUN pip3 install pytest mock

COPY pkg/workloads /src
COPY pkg/aggregators /aggregators
COPY pkg/transformers /transformers
COPY pkg/estimators /estimators

COPY images/test/run.sh /src/run.sh

WORKDIR /src

ENTRYPOINT ["/bin/bash"]
CMD ["/src/run.sh"]
