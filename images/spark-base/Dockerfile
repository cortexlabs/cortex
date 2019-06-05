FROM ubuntu:16.04 as builder

RUN apt-get update -qq && apt-get install -y -q \
        git \
        curl \
        wget \
        openjdk-8-jdk \
        maven \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt

ARG HADOOP_VERSION="2.9.2"
ARG SPARK_VERSION="2.4.2"
ARG TF_VERSION="1.13.1"
# Required for building tensorflow spark connector
ARG SCALA_VERSION="2.12"
# Scalatest version from https://github.com/apache/spark/blob/v2.4.2/pom.xml
ARG SCALATEST_VERSION="3.0.3"
# Check aws-java-sdk-bundle dependency version: https://mvnrepository.com/artifact/org.apache.hadoop/hadoop-aws/$HADOOP_VERSION
ARG AWS_JAVA_SDK_VERSION="1.11.199"

ENV JAVA_HOME="/usr/lib/jvm/java-8-openjdk-amd64"
ENV HADOOP_HOME="/opt/hadoop"
ENV SPARK_HOME="/opt/spark"

# Hadoop (code for e.g. finding a mirror, checking the checksum: https://github.com/smizy/docker-hadoop-base/blob/master/Dockerfile, https://github.com/liubin/docker-hadoop/blob/master/Dockerfile)
RUN curl https://archive.apache.org/dist/hadoop/common/hadoop-${HADOOP_VERSION}/hadoop-${HADOOP_VERSION}.tar.gz | tar -zx && \
    mv hadoop-${HADOOP_VERSION} $HADOOP_HOME && \
    rm -rf $HADOOP_HOME/share/doc

# Spark
RUN curl https://archive.apache.org/dist/spark/spark-${SPARK_VERSION}/spark-${SPARK_VERSION}-bin-without-hadoop.tgz | tar -zx && \
    mv spark-${SPARK_VERSION}-bin-without-hadoop $SPARK_HOME

# Tensorflow Spark connector
RUN rm -rf ~/tf-ecosystem && git clone https://github.com/tensorflow/ecosystem.git ~/tf-ecosystem && \
    mvn -f ~/tf-ecosystem/hadoop/pom.xml versions:set -DnewVersion=${TF_VERSION} -q && \
    mvn -f ~/tf-ecosystem/hadoop/pom.xml -Dmaven.test.skip=true clean install -q && \
    mvn -f ~/tf-ecosystem/spark/spark-tensorflow-connector/pom.xml versions:set -DnewVersion=${TF_VERSION} -q && \
    mvn -f ~/tf-ecosystem/spark/spark-tensorflow-connector/pom.xml -Dmaven.test.skip=true clean install \
        -Dspark.version=${SPARK_VERSION} -Dscala.binary.version=${SCALA_VERSION} -Dscala.test.version=${SCALATEST_VERSION} -q && \
    mv ~/tf-ecosystem/spark/spark-tensorflow-connector/target/spark-tensorflow-connector_2.11-${TF_VERSION}.jar $SPARK_HOME/jars/

# Hadoop AWS
RUN wget -q -P $SPARK_HOME/jars/ http://central.maven.org/maven2/org/apache/hadoop/hadoop-aws/${HADOOP_VERSION}/hadoop-aws-${HADOOP_VERSION}.jar

# AWS SDK
RUN wget -q -P $SPARK_HOME/jars/ http://central.maven.org/maven2/com/amazonaws/aws-java-sdk-bundle/${AWS_JAVA_SDK_VERSION}/aws-java-sdk-bundle-${AWS_JAVA_SDK_VERSION}.jar

# Configuration
COPY images/spark-base/conf/* $SPARK_HOME/conf/


FROM ubuntu:16.04

ENV JAVA_HOME="/usr/lib/jvm/java-8-openjdk-amd64"
ENV HADOOP_HOME="/opt/hadoop"
ENV SPARK_HOME="/opt/spark"

ENV PATH="${PATH}:${HADOOP_HOME}/lib/native"
ENV HADOOP_OPTS="${HADOOP_OPTS} -Djava.library.path=${HADOOP_HOME}/lib/native"
ENV LD_LIBRARY_PATH="${HADOOP_HOME}/lib/native/:${LD_LIBRARY_PATH}"
ENV HADOOP_COMMON_LIB_NATIVE_DIR="${HADOOP_HOME}/lib/native"

RUN set -ex && \
    apt-get update -qq && \
    apt-get install -y -q \
        git \
        bash \
        openjdk-8-jre \
        cmake \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/* && \
    mkdir -p $SPARK_HOME/work-dir && \
    touch $SPARK_HOME/RELEASE && \
    rm /bin/sh && \
    ln -sv /bin/bash /bin/sh && \
    chgrp root /etc/passwd && chmod ug+rw /etc/passwd

COPY --from=builder $HADOOP_HOME $HADOOP_HOME
COPY --from=builder $SPARK_HOME $SPARK_HOME

RUN mkdir ~/tini && cd ~/tini && git clone https://github.com/krallin/tini.git ~/tini && \
    CFLAGS="-DPR_SET_CHILD_SUBREAPER=36 -DPR_GET_CHILD_SUBREAPER=37" cmake . && make && mv tini /sbin/tini
