FROM tensorflow/serving:2.1.0

COPY images/tensorflow-serving-cpu/run.sh /src/
RUN chmod +x /src/run.sh

ENTRYPOINT ["/src/run.sh"]
