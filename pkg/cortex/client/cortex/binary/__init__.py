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

import os
import base64
import sys
import subprocess
from typing import List
from cortex.exceptions import CortexBinaryException

MIXED_CORTEX_MARKER = "~~cortex~~"


def run():
    """
    Runs the CLI from terminal.
    """
    try:
        process = subprocess.run([get_cli_path()] + sys.argv[1:], cwd=os.getcwd())
    except KeyboardInterrupt:
        sys.exit(130)  # Ctrl + C
    sys.exit(process.returncode)


def run_cli(
    args: List[str],
    hide_output: bool = False,
    mixed_output: bool = False,
) -> str:
    """
    Runs the Cortex binary with the specified arguments.

    Args:
        args: Arguments to use when invoking the Cortex CLI.
        hide_output: Flag to prevent streaming CLI output to stdout.
        mixed_output: Used to handle CLI output that both prints to stdout and should be returned.

    Raises:
        CortexBinaryException: Cortex CLI command returned an error.

    Returns:
        The stdout from the Cortex CLI command, or the result if mixed_output output.
    """

    env = os.environ.copy()
    env["CORTEX_CLI_INVOKER"] = "python"
    process = subprocess.Popen(
        [get_cli_path()] + args,
        stderr=subprocess.PIPE,
        stdout=subprocess.PIPE,
        encoding="utf8",
        env=env,
    )

    output = ""
    result = ""
    processing_result = False
    processed_result = False

    for c in iter(lambda: process.stdout.read(1), ""):
        output += c

        if mixed_output:
            if output[-2:] == "\n~" or output == "~":
                processing_result = True
                output = output[:-1]
            if processing_result:
                result += c
                if (
                    result[: len(MIXED_CORTEX_MARKER)] == MIXED_CORTEX_MARKER
                    and result[-len(MIXED_CORTEX_MARKER) :] == MIXED_CORTEX_MARKER
                    and len(result) > len(MIXED_CORTEX_MARKER)
                ):
                    result = result[len(MIXED_CORTEX_MARKER) : -len(MIXED_CORTEX_MARKER)]
                    result = base64.b64decode(result).decode("utf8")
                    processed_result = True

                output = output[:-1]
        if not hide_output:
            if (not mixed_output) or (mixed_output and not processing_result):
                sys.stdout.write(c)
                sys.stdout.flush()

        if processed_result == True:
            processing_result = False

    process.wait()

    if process.returncode == 0:
        if mixed_output:
            return result
        return output

    if result != "":
        raise CortexBinaryException(result + "\n" + process.stderr.read())

    raise CortexBinaryException(process.stderr.read())


def get_cli_path() -> str:
    """
    Get the location of the CLI.

    Default location is the directory containing the `cortex.binary` package.
    The location can be overridden by setting the `CORTEX_CLI_PATH` environment variable.

    Raises:
        Exception: Unable to find the CLI.

    Returns:
        str: The location of the CLI in the local filesystem.
    """
    if os.environ.get("CORTEX_CLI_PATH") is not None:
        cli_path = os.environ["CORTEX_CLI_PATH"]
        if not os.path.exists(cli_path):
            raise Exception(
                f"unable to find cortex binary at {cli_path} as specified in `CORTEX_CLI_PATH` environment variable"
            )
        return cli_path

    try:
        import importlib.resources as pkg_resources
    except ImportError:
        # Try backported to PY<37 `importlib_resources`.
        import importlib_resources as pkg_resources

    from cortex import binary

    try:
        with pkg_resources.path(binary, "cli") as p:
            cli_path = p
    except FileNotFoundError as e:
        raise Exception(
            "unable to find cortex binary, please reinstall the cortex client by running `pip uninstall cortex` and then `pip install cortex`"
        ) from e

    return str(cli_path)
