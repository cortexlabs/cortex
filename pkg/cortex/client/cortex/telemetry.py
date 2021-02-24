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
import pathlib

from uuid import uuid4

import sentry_sdk
from sentry_sdk.integrations.dedupe import DedupeIntegration
from sentry_sdk.integrations.stdlib import StdlibIntegration
from sentry_sdk.integrations.modules import ModulesIntegration

from cortex.exceptions import CortexBinaryException
from cortex.consts import (
    CORTEX_VERSION,
    CORTEX_TELEMETRY_SENTRY_DSN,
    CORTEX_TELEMETRY_SENTRY_ENVIRONMENT,
)


def _sentry_client(
    disabled: bool = False,
) -> sentry_sdk.Client:
    """
    Initialize sentry. You can override the default values with the following env vars:
    1. CORTEX_TELEMETRY_SENTRY_DSN
    2. CORTEX_TELEMETRY_SENTRY_ENVIRONMENT
    3. CORTEX_TELEMETRY_DISABLE
    """

    dsn = CORTEX_TELEMETRY_SENTRY_DSN
    environment = CORTEX_TELEMETRY_SENTRY_ENVIRONMENT

    if disabled or os.getenv("CORTEX_TELEMETRY_DISABLE", "").lower() == "true":
        return

    if os.getenv("CORTEX_TELEMETRY_SENTRY_DSN", "") != "":
        dsn = os.environ["CORTEX_TELEMETRY_SENTRY_DSN"]

    if os.getenv("CORTEX_TELEMETRY_SENTRY_ENVIRONMENT", "") != "":
        environment = os.environ["CORTEX_TELEMETRY_SENTRY_ENVIRONMENT"]

    client = sentry_sdk.Client(
        dsn=dsn,
        environment=environment,
        release=CORTEX_VERSION,
        ignore_errors=[CortexBinaryException],  # exclude CortexBinaryException exceptions
        in_app_include=["cortex"],  # for better grouping of events in sentry
        attach_stacktrace=True,
        default_integrations=False,  # disable all default integrations
        auto_enabling_integrations=False,
        integrations=[
            DedupeIntegration(),  # prevent duplication of events
            StdlibIntegration(),  # adds breadcrumbs (aka more info)
            ModulesIntegration(),  # adds info about installed modules
        ],
        # debug=True,
    )

    return client


def _create_default_scope(optional_tags: dict = {}) -> sentry_sdk.Scope:
    """
    Creates default scope. Adds user ID as tag to the reported event.
    Can add optional tags.
    """

    scope = sentry_sdk.Scope()

    user_id = None
    client_id_file_path = pathlib.Path.home() / ".cortex" / "client-id.txt"
    if not client_id_file_path.is_file():
        client_id_file_path.write_text(str(uuid4()))
    user_id = client_id_file_path.read_text()

    if user_id:
        scope.set_user({"id": user_id})

    for k, v in optional_tags.items():
        scope.set_tag(k, v)

    return scope


# only one instance of this is required
hub = sentry_sdk.Hub(_sentry_client(), _create_default_scope())


def sentry_wrapper(func):
    def wrapper(*args, **kwargs):
        with hub:
            try:
                return func(*args, **kwargs)
            except:
                sentry_sdk.capture_exception()
                sentry_sdk.flush()
                raise

    return wrapper
