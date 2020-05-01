# NVIDIA container runtime not found

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When attempting to deploy a model to a GPU in local environment, you might encounter NVIDIA container runtime not found. Cortex deploys APIs in the local environment using Docker. You can deploy models on a GPU using Cortex in a local environment if your Docker engine has the NVIDIA container runtime. THe NVIDIA container runtime is responsible for exposing your GPU to the Docker engine.

## Check Compatibility

Please ensure that your local machine has an NVIDIA GPU card installed.

Mac OSX and Windows are currently not supported by NVIDIA container runtime. You can find the complete list of supported operating system and architectures [here](https://nvidia.github.io/nvidia-container-runtime/).

## Install NVIDIA container runtime

Instructions for setting up NVIDIA container runtime can be found [here](https://github.com/NVIDIA/nvidia-container-runtime#installation).

You can verify that the NVIDIA container runtime has been installed successfully if `nvidia` is listed in the available runtimes `docker info | grep -i runtime`.
