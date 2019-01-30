#!/bin/bash

# Copyright 2019 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


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
