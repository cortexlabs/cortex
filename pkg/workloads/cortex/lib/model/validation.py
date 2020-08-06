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

import os
from typing import List

from cortex.lib.storage import S3, LocalStorage
from cortex.lib.log import cx_logger
from cortex.lib.exceptions import CortexException
from cortex.lib.model import ModelsTree
from cortex.lib.api import (
    PythonPredictorType,
    TensorFlowPredictorType,
    ONNXPredictorType,
    PredictorType,
)

import collections


class TemplatePlaceholder(collections.namedtuple("TemplatePlaceholder", "placeholder")):
    """
    Placeholder type that denotes an operation, a text placeholder, etc.
    """

    def __new__(cls, placeholder: str):
        return super(cls, TemplatePlaceholder).__new__(cls, "<" + placeholder + ">")

    def __str__(self) -> str:
        return str(self.placeholder)

    def __repr__(self) -> str:
        return str(self.placeholder)

    @property
    def type(self) -> str:
        return str(self.placeholder).strip("<>")


class GenericPlaceholder(collections.namedtuple("GenericPlaceholder", "placeholder value")):
    """
    Generic placeholder.

    Can hold any value.
    Can be of one type only: generic.

    Accessible properties: value and placeholder.
    """

    def __new__(cls, value: str):
        return super(cls, GenericPlaceholder).__new__(cls, "<generic>", value)

    def __eq__(self, other) -> bool:
        if isinstance(other, GenericPlaceholder):
            return self.placeholder == other.placeholder
        return False

    def __hash__(self):
        return hash((self.placeholder, self.value))

    def __str__(self) -> str:
        return str(self.value)

    def __repr__(self) -> str:
        return str(self.value)

    @property
    def type(self) -> str:
        return str(self.placeholder).strip("<>")


class PlaceholderGroup:
    """
    Order-based addition of placeholder types (Groups, Generics or Templates).
    """

    def __init__(self, *args):
        self.parts = args

    def __getitem__(self, index: int):
        return self.parts[index]

    def __len__(self) -> int:
        return len(self.parts)

    def __str__(self) -> str:
        return str(self.parts)

    def __repr__(self) -> str:
        return str(self.parts)


IntegerPlaceholder = TemplatePlaceholder("integer")  # the path name must be an integer
AnyPlaceholder = TemplatePlaceholder(
    "any"
)  # the path can be any file or any directory (with multiple subdirectories)
SinglePlaceholder = TemplatePlaceholder(
    "single"
)  # can only have a single occurrence of this, but its name can take any form
ExclAlternativePlaceholder = TemplatePlaceholder(
    "exclusive"
)  # can either be this template xor anything else at the same level


# to be used when predictor:model_path or predictor:models:paths is used
model_template = {
    PythonPredictorType: {IntegerPlaceholder: AnyPlaceholder},
    TensorFlowPredictorType: {
        IntegerPlaceholder: {
            AnyPlaceholder: None,
            GenericPlaceholder("saved_model.pb"): None,
            GenericPlaceholder("variables"): {
                GenericPlaceholder("variables.index"): None,
                PlaceholderGroup(
                    GenericPlaceholder("variables.data-00000-of-"), AnyPlaceholder
                ): None,
                AnyPlaceholder: None,
            },
        },
    },
    ONNXPredictorType: {
        IntegerPlaceholder: {
            PlaceholderGroup(SinglePlaceholder, GenericPlaceholder(".onnx")): None,
        },
        PlaceholderGroup(
            ExclAlternativePlaceholder, SinglePlaceholder, GenericPlaceholder(".onnx")
        ): None,
    },
}

CustomPredictorType = PredictorType("custom")
model_template = {CustomPredictorType: {IntegerPlaceholder: None, AnyPlaceholder: None}}


def json_model_template_representation(model_template) -> dict:
    dct = {}
    if model_template is None:
        return None
    if isinstance(model_template, dict):
        for key in model_template:
            dct[str(key)] = json_model_template_representation(model_template[key])
        return dct
    else:
        return str(model_template)


def dir_models_pattern(predictor_type: PredictorType) -> dict:
    """
    To be used when predictor:models:dir in cortex.yaml is used.
    """
    return {SinglePlaceholder: model_template[predictor_type]}


def single_model_pattern(predictor_type: PredictorType) -> dict:
    """
    To be used when predictor:model_path or predictor:models:paths in cortex.yaml is used.
    """
    return model_template[predictor_type]


def validate_s3_models_dir_paths(s3_top_paths: List[str], predictor_type: PredictorType) -> list:
    """
    To be used when predictor:models:dir in cortex.yaml is used.
    """
    for s3_top_path in s3_top_paths:
        model_name = os.path.dirname(s3_top_path)
    return {}


def validate_s3_model_paths(
    s3_paths: List[str], predictor_type: PredictorType, commonprefix: str
) -> None:
    """
    To be used when predictor:model_path or predictor:models:paths in cortex.yaml is used.
    """
    if len(s3_paths) == 0:
        raise CortexException()

    pattern = single_model_pattern(predictor_type)
    keys = pattern.keys()
    paths = [os.path.relpath(s3_path, commonprefix) for s3_path in s3_paths]
    objects = [get_leftmost_part_of_path(path) for path in paths]
    objects = list(set(objects))
    visited_objects = len(objects) * [False]

    try:
        for key in keys:
            if key == IntegerPlaceholder:
                validate_integer_placeholder(keys, objects, visited_objects)
            elif key == AnyPlaceholder:
                validate_any_placeholder(keys, objects, visited_objects)
            elif key == SinglePlaceholder:
                validate_single_placeholder(keys, objects, visited_objects)
            elif key == GenericPlaceholder(""):
                validate_generic_placeholder(keys, objects, visited_objects, key.value)
            elif isinstance(key, PlaceholderGroup):
                validate_group_placeholder(keys, objects, visited_objects)
            elif key == ExclAlternativePlaceholder:
                validate_exclusive_placeholder(keys, objects, visited_objects)
            else:
                raise CortexException("found a non-placeholder object in model template")
    except CortexException as e:
        raise CortexException(f"{predictor_type} predictor at '{commonprefix}'", str(e))

    unvisited_paths = []
    for idx, visited in enumerate(visited_objects):
        if not visited:
            unvisited_paths.append(paths[idx])
    if len(unvisited_paths) > 0:
        raise CortexException(
            f"{predictor_type} predictor model at '{commonprefix}'",
            "unexpected path(s) for " + str(unvisited_paths),
        )


def get_leftmost_part_of_path(path: str) -> str:
    basename = ""
    while path:
        path, basename = os.path.split(path)
    return basename


def validate_integer_placeholder(
    placeholders: list, objects: List[str], visited: List[bool]
) -> None:
    appearances = 0
    for idx, obj in enumerate(objects):
        if obj.isnumeric():
            visited[idx] = True
            appearances += 1

    if appearances > 1 and len(placeholders) == 1:
        raise CortexException(f"too many {IntegerPlaceholder} appearances in path")


def validate_any_placeholder(placeholders: list, objects: List[str], visited: List[bool]) -> None:
    for idx in range(len(visited)):
        visited[idx] = True


def validate_single_placeholder(
    placeholders: list, objects: List[str], visited: List[bool]
) -> None:
    if len(placeholders) > 1 or len(objects) > 1:
        raise CortexException(f"only a single {SinglePlaceholder} is allowed per directory")
    if len(visited) > 0:
        visited[0] = True


def validate_generic_placeholder(
    placeholders: list, objects: List[str], visited: List[bool], generic_value: str
) -> None:
    for idx, obj in enumerate(objects):
        if obj == generic_value:
            visited[idx] = True
            break


def validate_group_placeholder(placeholders: list, objects: List[str], visited: List[bool]) -> None:
    pass


def validate_exclusive_placeholder(
    placeholders: list, objects: List[str], visited: List[bool]
) -> None:
    pass
