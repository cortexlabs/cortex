FROM nvcr.io/nvidia/k8s/dcgm-exporter:2.1.4-2.3.1-ubuntu18.04

# uncomment DCGM_FI_DEV_GPU_UTIL metric
RUN sed -i '/DCGM_FI_DEV_GPU_UTIL/s/^# //g' /etc/dcgm-exporter/default-counters.csv
