# Install CLI on Windows

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

This guide walks you through installing the Cortex CLI on a Windows 10 machine using the Linux Subsystem as its backend. 

The requirement is an x64 system with Windows 10 of **Version 1903** or higher, with **Build 18362** or higher.

## Step 1

Install and configure the WSL (Windows Subsystem for Linux) of version 2 on your machine following [this installation guide](https://docs.microsoft.com/en-us/windows/wsl/install-win10).

In our example, we assume the installation of the Ubuntu distribution.

## Step 2

Install and configure the Docker Desktop engine app to use the WSL (Windows Subsystem for Linux) backend of version 2. The instructions on how to do that are found in the [Docker Desktop WSL 2 backend guide](https://docs.docker.com/docker-for-windows/wsl/).

## Step 3

Run Ubuntu in the terminal on your Windows machine and right-click the window's bar and click on *Properties*.

![step-3a](https://user-images.githubusercontent.com/26958764/96926494-493cdf80-14be-11eb-9fac-4c81e1fac55c.png)

In the *Font* category, set the font to one of the following fonts: **MS Gothic**, **NSimSun** or **SimSun-ExtB**. The recommendation is to use **SimSun-ExtB**. Once selected, click *Okay*.

![step-3b](https://user-images.githubusercontent.com/26958764/96926763-adf83a00-14be-11eb-9584-4eff3faf2377.png)

Setting the font to any of the provided ones helps render all Unicode characters correctly.

## Step 4

Within the Ubuntu terminal, install the Cortex CLI as you'd do it on a Mac/Linux machine. The CLI commands are documented [here](../miscellaneous/cli.md#command-overview).

### Install the CLI with Python Client

```bash
pip install cortex
```

### Install the CLI without Python Client

```bash
# Replace `INSERT_CORTEX_VERSION` with the complete CLI version (e.g. 0.18.1):
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/vINSERT_CORTEX_VERSION/get-cli.sh)"

# For example to download CLI version 0.18.1 (Note the 'v'):
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.18.1/get-cli.sh)"
```

## Step 5

Start using the Cortex CLI.

![step-5](https://user-images.githubusercontent.com/26958764/96927485-ca48a680-14bf-11eb-909c-f15dbc52af08.png)
