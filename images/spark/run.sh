#!/bin/bash
set -e

# Use our conf (this env var was clobbered at some point, so reset it)
export SPARK_CONF_DIR=$SPARK_HOME/conf-custom

# Keep conf/spark.properties, which is generated and mounted by spark-on-k8s on the driver (KubernetesClientApplication.scala)
if [ -f $SPARK_HOME/conf/spark.properties ]; then
  cp $SPARK_HOME/conf/spark.properties $SPARK_HOME/conf-custom/spark.properties
fi

# entrypoint.sh doesn't do any of the following for executor (it does for driver):
. $SPARK_HOME/bin/load-spark-env.sh  # Load spark-env.sh
export SPARK_EXTRA_CLASSPATH=$($HADOOP_HOME/bin/hadoop classpath)  # Add hadoop to classpath
export SPARK_JAVA_OPT_99="-Dlog4j.configuration=file://${SPARK_HOME}/conf/log4j.properties"  # Add log config (99 is used to avoid conflicts)

echo ""
echo "Starting"
echo ""

# Run the intended command
/opt/entrypoint.sh "$@"
