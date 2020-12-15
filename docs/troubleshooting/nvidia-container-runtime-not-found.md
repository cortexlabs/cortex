# NVIDIA container runtime not found

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When attempting to deploy a model to a GPU in the local environment, you may encounter NVIDIA container runtime not found. Since Cortex uses Docker to deploy APIs in the local environment, your Docker engine must have the NVIDIA container runtime installed (the NVIDIA container runtime is responsible for exposing your GPU to the Docker engine).

## Check Compatibility

Please ensure that your local machine has an NVIDIA GPU card installed. Mac and Windows are currently not supported by the NVIDIA container runtime. You can find the complete list of supported operating system and architectures [here](https://nvidia.github.io/nvidia-container-runtime).

## Install NVIDIA container runtime

Instructions for setting up the NVIDIA container runtime can be found [here](https://github.com/NVIDIA/nvidia-container-runtime#installation).

You can verify that the NVIDIA container runtime has been installed successfully if `nvidia` is listed in the available runtimes: `docker info | grep -i runtime`.
