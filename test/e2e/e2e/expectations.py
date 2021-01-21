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

import types
from typing import Dict, Any

import jsonschema
import requests
import yaml
from jsonschema import Draft7Validator

from e2e.exceptions import ExpectationsValidationException

CONTENT_TO_ATTR = {"text": "text", "json": "json", "binary": "content"}


def assert_response_expectations(response: requests.Response, expectations: Dict[str, Any]):
    content_type = expectations["content_type"]

    expected = expectations.get("expected")
    if expected:
        output = _get_response_content(response, content_type)
        assert output == expected, f"unexpected response: got {output}, expected {expected}"

    expected_json_schema = expectations.get("json_schema")
    if expected_json_schema:
        output = _get_response_content(response, content_type)
        jsonschema.validate(output, schema=expected_json_schema)


def parse_expectations(expectations_file: str) -> Dict[str, Any]:
    with open(expectations_file) as f:
        expectations = yaml.safe_load(f)

    validate_expectations(expectations)

    return expectations


def validate_expectations(expectations):
    if "response" in expectations:
        validate_response_expectations(expectations["response"])


def validate_response_expectations(expectations: Dict[str, Any]):
    if not expectations["content_type"] in CONTENT_TO_ATTR.keys():
        raise ExpectationsValidationException(
            f"response.content_type should be one of {CONTENT_TO_ATTR.keys()}"
        )

    if "expected" in expectations and "json_schema" in expectations:
        raise ExpectationsValidationException("expected and json_schema are mutually exclusive")

    if "json_schema" in expectations:
        if expectations["content_type"] != "json":
            raise ExpectationsValidationException(
                "json_schema is only valid when content_type is set to json"
            )

        try:
            Draft7Validator.check_schema(schema=expectations["json_schema"])
        except Exception as e:
            raise ExpectationsValidationException("json_schema is invalid") from e


def _get_response_content(response: requests.Response, content_type: str) -> str:
    attr = CONTENT_TO_ATTR.get(content_type, "content")
    content = getattr(response, attr)
    if isinstance(content, types.MethodType):
        return content()

    return content
