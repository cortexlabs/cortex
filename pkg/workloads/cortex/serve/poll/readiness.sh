#!/usr/bin/with-contenv bash

while true; do
    procs_ready="$(ls /mnt/workspace/proc-*-ready.txt 2>/dev/null | wc -l)"
    if [ "$CORTEX_PROCESSES_PER_REPLICA" = "$procs_ready" ]; then
        touch /mnt/workspace/api_readiness.txt
        break
    fi
done
