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
from typing import Optional

import sentry_sdk

import cortex_internal.lib.exceptions as cexp
from sentry_sdk.integrations.logging import LoggingIntegration
from sentry_sdk.integrations.threading import ThreadingIntegration


def get_default_tags():
    vars = {
        "provider": "CORTEX_PROVIDER",
        "kind": "CORTEX_KIND",
        "image_type": "CORTEX_IMAGE_TYPE",
    }

    tags = {}
    for target_tag, env_var in vars.items():
        value = os.getenv(env_var)
        if value and value != "":
            tags[target_tag] = value

    return tags


def init_sentry(
    dsn: str = "",
    environment: str = "",
    release: str = "",
    tags: dict = {},
    disabled: Optional[bool] = None,
):
    """
    Initialize sentry. If no arguments are passed in, the following env vars will be used instead.
    1. dsn -> CORTEX_TELEMETRY_SENTRY_DSN
    2. environment -> CORTEX_TELEMETRY_SENTRY_ENVIRONMENT
    3. release -> CORTEX_VERSION
    4. disabled -> CORTEX_TELEMETRY_DISABLE

    In addition to that, the user ID tag is added to every reported event.
    """

    if disabled is True or os.getenv("CORTEX_TELEMETRY_DISABLE", "").lower() == "true":
        return

    if dsn == "":
        dsn = os.environ["CORTEX_TELEMETRY_SENTRY_DSN"]

    if environment == "":
        environment = os.environ["CORTEX_TELEMETRY_SENTRY_ENVIRONMENT"]

    if release == "":
        release = os.environ["CORTEX_VERSION"]

    user_id = os.environ["CORTEX_TELEMETRY_SENTRY_USER_ID"]

    sentry_sdk.init(
        dsn=dsn,
        environment=environment,
        release=release,
        ignore_errors=[cexp.UserException, cexp.UserRuntimeException],
        attach_stacktrace=True,
        integrations=[
            LoggingIntegration(event_level=None, level=None),
            ThreadingIntegration(propagate_hub=True),
        ],
    )
    sentry_sdk.set_user({"id": user_id})

    for k, v in tags.items():
        sentry_sdk.set_tag(k, v)


def capture_exception(err: Exception, level: str = "error", extra_tags: dict = {}):
    """
    Capture an exception. Optionally set the log level of the event. Extra tags accepted.
    """

    with sentry_sdk.push_scope() as scope:
        scope.set_level(level)
        for k, v in extra_tags:
            scope.set_tag(k, v)
        sentry_sdk.capture_exception(err, scope)
        sentry_sdk.flush()
