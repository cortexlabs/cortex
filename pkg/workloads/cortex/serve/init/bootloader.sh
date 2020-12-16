#!/usr/bin/with-contenv bash

# Copyright 2020 Cortex Labs, Inc.
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
    if [ "$CORTEX_PROVIDER" == "local" ]; then
        echo "error: your Cortex CLI version ($CORTEX_VERSION) doesn't match your predictor image version ($EXPECTED_CORTEX_VERSION); please update your predictor image by modifying the \`image\` field in your API configuration file (e.g. cortex.yaml) and re-running \`cortex deploy\`, or update your CLI by following the instructions at https://docs.cortex.dev/"
    else
        echo "error: your Cortex operator version ($CORTEX_VERSION) doesn't match your predictor image version ($EXPECTED_CORTEX_VERSION); please update your predictor image by modifying the \`image\` field in your API configuration file (e.g. cortex.yaml) and re-running \`cortex deploy\`, or update your cluster by following the instructions at https://docs.cortex.dev/"
    fi
    exit 1
fi

mkdir -p /mnt/workspace
mkdir -p /mnt/requests

cd /mnt/project

# if the container restarted, ensure that it is not perceived as ready
rm -rf /mnt/workspace/api_readiness.txt
rm -rf /mnt/workspace/init_script_run.txt
rm -rf /mnt/workspace/proc-*-ready.txt

# allow for the liveness check to pass until the API is running
echo "9999999999" > /mnt/workspace/api_liveness.txt

# to export user-specified environment files
source_env_file_cmd="if [ -f \"/mnt/project/.env\" ]; then set -a; source /mnt/project/.env; set +a; fi"

if [ "$CORTEX_PROVIDER" != "local" ]; then
    if [ "$CORTEX_KIND" == "RealtimeAPI" ]; then
        sysctl -w net.core.somaxconn="65535" >/dev/null
        sysctl -w net.ipv4.ip_local_port_range="15000 64000" >/dev/null
        sysctl -w net.ipv4.tcp_fin_timeout=30 >/dev/null
    fi
fi

# execute script if present in project's directory
if [ -f "/mnt/project/dependencies.sh" ]; then
    eval $source_env_file_cmd
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

        pip --no-cache-dir install -r /src/cortex/serve/requirements.txt
        if [ -f "/src/cortex/serve/image.requirements.txt" ]; then
            pip --no-cache-dir install -r /src/cortex/serve/image.requirements.txt
        fi

        rm -rf $CONDA_PREFIX/lib/python${old_py_version}  # previous python is no longer needed
    fi
fi

# install pip packages
if [ -f "/mnt/project/requirements.txt" ]; then
    pip --no-cache-dir install -r /mnt/project/requirements.txt
fi

create_s6_service() {
    # good pages to read about s6-overlay
    # https://wiki.gentoo.org/wiki/S6#Process_supervision
    # https://skarnet.org/software/s6/s6-svscanctl.html
    # http://skarnet.org/software/s6/s6-svc.html
    # http://skarnet.org/software/s6/servicedir.html

    # good pages to read about execline
    # http://www.troubleshooters.com/linux/execline.htm
    # https://danyspin97.org/blog/getting-started-with-execline-scripting/

    service_name=$1
    cmd=$2

    dest_dir="/etc/services.d/$service_name"
    mkdir $dest_dir

    dest_script="$dest_dir/run"
    echo "#!/usr/bin/with-contenv bash" > $dest_script
    echo $cmd >> $dest_script
    chmod +x $dest_script

    dest_script="$dest_dir/finish"
    echo "#!/usr/bin/execlineb -S0" > $dest_script
    echo "ifelse { s6-test \${1} -ne 0 } { foreground { redirfd -w 1 /var/run/s6/env-stage3/S6_STAGE2_EXITED s6-echo -n -- \${1} } s6-svscanctl -t /var/run/s6/services }" >> $dest_script
    echo "s6-svc -O /var/run/s6/services/$service_name" >> $dest_script
    chmod +x $dest_script
}

# prepare webserver
if [ "$CORTEX_KIND" = "RealtimeAPI" ]; then

    # prepare uvicorn workers
    mkdir /run/uvicorn
    for i in $(seq 1 $CORTEX_PROCESSES_PER_REPLICA); do
        create_s6_service "uvicorn-$((i-1))" "cd /mnt/project && $source_env_file_cmd && exec env PYTHONUNBUFFERED=TRUE env PYTHONPATH=$PYTHONPATH:$CORTEX_PYTHON_PATH /opt/conda/envs/env/bin/python /src/cortex/serve/start/server.py /run/uvicorn/proc-$((i-1)).sock"
    done

    create_s6_service "nginx" "exec nginx -c /run/nginx.conf"

    # prepare api readiness checker
    dest_dir="/etc/services.d/api_readiness"
    mkdir $dest_dir
    cp /src/cortex/serve/poll/readiness.sh $dest_dir/run
    chmod +x $dest_dir/run

    # generate nginx conf
    /opt/conda/envs/env/bin/python -c 'from cortex.lib import util; import os; generated = util.render_jinja_template("/src/cortex/serve/nginx.conf.j2", os.environ); print(generated);' > /run/nginx.conf

    # create the python initialization service
    create_s6_service "py_init" "cd /mnt/project && /opt/conda/envs/env/bin/python /src/cortex/serve/init/script.py"
elif [ "$CORTEX_KIND" = "BatchAPI" ]; then
    create_s6_service "batch" "cd /mnt/project && $source_env_file_cmd && exec env PYTHONUNBUFFERED=TRUE env PYTHONPATH=$PYTHONPATH:$CORTEX_PYTHON_PATH /opt/conda/envs/env/bin/python /src/cortex/serve/start/batch.py"

    # create the python initialization service
    create_s6_service "py_init" "cd /mnt/project && /opt/conda/envs/env/bin/python /src/cortex/serve/init/script.py"
elif [ "$CORTEX_KIND" = "TaskAPI" ]; then
    create_s6_service "task" "cd /mnt/project && $source_env_file_cmd && exec env PYTHONUNBUFFERED=TRUE env PYTHONPATH=$PYTHONPATH:$CORTEX_PYTHON_PATH /opt/conda/envs/env/bin/python /src/cortex/serve/start/task.py"
fi




