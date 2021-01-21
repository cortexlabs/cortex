#!/usr/bin/with-contenv bash

# Copyright 2021 Cortex Labs, Inc.
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

# CORTEX_VERSION
export EXPECTED_CORTEX_VERSION=master

if [ "$CORTEX_VERSION" != "$EXPECTED_CORTEX_VERSION" ]; then
    echo "error: your Cortex operator version ($CORTEX_VERSION) doesn't match your predictor image version ($EXPECTED_CORTEX_VERSION); please update your predictor image by modifying the \`image\` field in your API configuration file (e.g. cortex.yaml) and re-running \`cortex deploy\`, or update your cluster by following the instructions at https://docs.cortex.dev/"
    exit 1
fi

function substitute_env_vars() {
    file_to_run_substitution=$1
    /opt/conda/envs/env/bin/python -c "from cortex_internal.lib import util; import os; util.expand_environment_vars_on_file('$file_to_run_substitution')"
}

# configure log level for python scripts§
substitute_env_vars $CORTEX_LOG_CONFIG_FILE

mkdir -p /mnt/workspace
mkdir -p /mnt/requests

cd /mnt/project

# if the container restarted, ensure that it is not perceived as ready
rm -rf /mnt/workspace/api_readiness.txt
rm -rf /mnt/workspace/init_script_run.txt
rm -rf /mnt/workspace/proc-*-ready.txt

if [ "$CORTEX_KIND" == "RealtimeAPI" ]; then
    sysctl -w net.core.somaxconn="65535" >/dev/null
    sysctl -w net.ipv4.ip_local_port_range="15000 64000" >/dev/null
    sysctl -w net.ipv4.tcp_fin_timeout=30 >/dev/null
fi

# to export user-specified environment files
source_env_file_cmd="if [ -f \"/mnt/project/.env\" ]; then set -a; source /mnt/project/.env; set +a; fi"

function install_deps() {
    eval $source_env_file_cmd

    # execute script if present in project's directory
    if [ -f "/mnt/project/dependencies.sh" ]; then
        bash -e /mnt/project/dependencies.sh
    fi

    # install from conda-packages.txt
    if [ -f "/mnt/project/conda-packages.txt" ]; then
        py_version_cmd='echo $(python -c "import sys; v=sys.version_info[:2]; print(\"{}.{}\".format(*v));")'
        old_py_version=$(eval $py_version_cmd)

        # look for packages in defaults and then conda-forge to improve chances of finding the package (specifically for python reinstalls)
        conda config --append channels conda-forge

        conda install -y --file /mnt/project/conda-packages.txt

        new_py_version=$(eval $py_version_cmd)

        # reinstall core packages if Python version has changed
        if [ $old_py_version != $new_py_version ]; then
            echo "warning: you have changed the Python version from $old_py_version to $new_py_version; this may break Cortex's web server"
            echo "reinstalling core packages ..."

            pip --no-cache-dir install \
                -r /src/cortex/serve/serve.requirements.txt \
                /src/cortex/serve/
            if [ -f "/src/cortex/serve/image.requirements.txt" ]; then
                pip --no-cache-dir install -r /src/cortex/serve/image.requirements.txt
            fi
        fi
    fi

    # install pip packages
    if [ -f "/mnt/project/requirements.txt" ]; then
        pip --no-cache-dir install -r /mnt/project/requirements.txt
    fi

}

# install user dependencies
if [ "$CORTEX_LOG_LEVEL" = "DEBUG" ] || [ "$CORTEX_LOG_LEVEL" = "INFO" ]; then
    install_deps
# if log level is set to warning/error
else
    # buffer install_deps stdout/stderr to a file
    tempf=$(mktemp)
    set +e
    (
        set -e
        install_deps
    ) > $tempf 2>&1
    set -e

    # if there was an error while running install_deps
    # print the stdout/stderr and exit
    exit_code=$?
    if [ $exit_code -ne 0 ]; then
        cat $tempf
        exit $exit_code
    fi
    rm $tempf
fi

# good pages to read about s6-overlay used in create_s6_service and create_s6_task
# https://wiki.gentoo.org/wiki/S6#Process_supervision
# https://skarnet.org/software/s6/s6-svscanctl.html
# http://skarnet.org/software/s6/s6-svc.html
# http://skarnet.org/software/s6/servicedir.html

# good pages to read about execline
# http://www.troubleshooters.com/linux/execline.htm
# https://danyspin97.org/blog/getting-started-with-execline-scripting/

# only terminate pod if this process exits with non-zero exit code
create_s6_service() {
    export SERVICE_NAME=$1
    export COMMAND_TO_RUN=$2

    dest_dir="/etc/services.d/$SERVICE_NAME"
    mkdir $dest_dir

    dest_script="$dest_dir/run"
    cp /src/cortex/serve/init/templates/run $dest_script
    substitute_env_vars $dest_script
    chmod +x $dest_script

    dest_script="$dest_dir/finish"
    cp /src/cortex/serve/init/templates/service_finish $dest_script
    substitute_env_vars $dest_script
    chmod +x $dest_script

    unset SERVICE_NAME
    unset COMMAND_TO_RUN
}

# terminate pod if this process exits (zero or non-zero exit code)
create_s6_task() {
    export TASK_NAME=$1
    export COMMAND_TO_RUN=$2

    dest_dir="/etc/services.d/$TASK_NAME"
    mkdir $dest_dir

    dest_script="$dest_dir/run"
    cp /src/cortex/serve/init/templates/run $dest_script
    substitute_env_vars $dest_script
    chmod +x $dest_script

    dest_script="$dest_dir/finish"
    cp /src/cortex/serve/init/templates/task_finish $dest_script
    substitute_env_vars $dest_script
    chmod +x $dest_script

    unset TASK_NAME
    unset COMMAND_TO_RUN
}

# prepare webserver
if [ "$CORTEX_KIND" = "RealtimeAPI" ]; then

    # prepare uvicorn workers
    mkdir /run/uvicorn
    for i in $(seq 1 $CORTEX_PROCESSES_PER_REPLICA); do
        create_s6_service "uvicorn-$((i-1))" "cd /mnt/project && $source_env_file_cmd && PYTHONUNBUFFERED=TRUE PYTHONPATH=$PYTHONPATH:$CORTEX_PYTHON_PATH exec /opt/conda/envs/env/bin/python /src/cortex/serve/start/server.py /run/uvicorn/proc-$((i-1)).sock"
    done

    create_s6_service "nginx" "exec nginx -c /run/nginx.conf"

    # prepare api readiness checker
    dest_dir="/etc/services.d/api_readiness"
    mkdir $dest_dir
    cp /src/cortex/serve/poll/readiness.sh $dest_dir/run
    chmod +x $dest_dir/run

    # generate nginx conf
    /opt/conda/envs/env/bin/python -c 'from cortex_internal.lib import util; import os; generated = util.render_jinja_template("/src/cortex/serve/nginx.conf.j2", os.environ); print(generated);' > /run/nginx.conf

    # create the python initialization service
    create_s6_service "py_init" "cd /mnt/project && exec /opt/conda/envs/env/bin/python /src/cortex/serve/init/script.py"
elif [ "$CORTEX_KIND" = "BatchAPI" ]; then
    create_s6_task "job" "cd /mnt/project && $source_env_file_cmd && PYTHONUNBUFFERED=TRUE PYTHONPATH=$PYTHONPATH:$CORTEX_PYTHON_PATH exec /opt/conda/envs/env/bin/python /src/cortex/serve/start/batch.py"

    # create the python initialization service
    create_s6_service "py_init" "cd /mnt/project && /opt/conda/envs/env/bin/python /src/cortex/serve/init/script.py"
elif [ "$CORTEX_KIND" = "TaskAPI" ]; then
    create_s6_task "job" "cd /mnt/project && $source_env_file_cmd && PYTHONUNBUFFERED=TRUE PYTHONPATH=$PYTHONPATH:$CORTEX_PYTHON_PATH exec /opt/conda/envs/env/bin/python /src/cortex/serve/start/task.py"
fi
