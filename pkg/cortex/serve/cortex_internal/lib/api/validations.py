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

import inspect
from typing import Dict, List

from cortex_internal.lib import util
from cortex_internal.lib.exceptions import UserException
from cortex_internal.lib.type import handler_type_from_api_spec, PythonHandlerType


def validate_class_impl(impl, impl_req):
    for optional_func_signature in impl_req.get("optional", []):
        validate_optional_method_args(impl, optional_func_signature)

    for required_func_signature in impl_req.get("required", []):
        validate_required_method_args(impl, required_func_signature)


def validate_optional_method_args(impl, func_signature):
    if getattr(impl, func_signature["name"], None):
        validate_required_method_args(impl, func_signature)


def validate_required_method_args(impl, func_signature):
    target_class_name = impl.__name__

    fn = getattr(impl, func_signature["name"], None)
    if not fn:
        raise UserException(
            f"class {target_class_name}",
            f'required method "{func_signature["name"]}" is not defined',
        )

    if not callable(fn):
        raise UserException(
            f"class {target_class_name}",
            f'"{func_signature["name"]}" is defined, but is not a method',
        )

    required_args = func_signature.get("required_args", [])
    optional_args = func_signature.get("optional_args", [])

    argspec = inspect.getfullargspec(fn)
    fn_str = f'{func_signature["name"]}({", ".join(argspec.args)})'

    for arg_name in required_args:
        if arg_name not in argspec.args:
            raise UserException(
                f"class {target_class_name}",
                f'invalid signature for method "{fn_str}"',
                f'"{arg_name}" is a required argument, but was not provided',
            )

        if arg_name == "self":
            if argspec.args[0] != "self":
                raise UserException(
                    f"class {target_class_name}",
                    f'invalid signature for method "{fn_str}"',
                    f'"self" must be the first argument',
                )

    seen_args = []
    for arg_name in argspec.args:
        if arg_name not in required_args and arg_name not in optional_args:
            raise UserException(
                f"class {target_class_name}",
                f'invalid signature for method "{fn_str}"',
                f'"{arg_name}" is not a supported argument',
            )

        if arg_name in seen_args:
            raise UserException(
                f"class {target_class_name}",
                f'invalid signature for method "{fn_str}"',
                f'"{arg_name}" is duplicated',
            )

        seen_args.append(arg_name)


def validate_python_handler_with_models(impl, api_spec):
    if not are_models_specified(api_spec):
        return

    target_class_name = impl.__name__
    constructor = getattr(impl, "__init__")
    constructor_arg_spec = inspect.getfullargspec(constructor)
    if "model_client" not in constructor_arg_spec.args:
        raise UserException(
            f"class {target_class_name}",
            f'invalid signature for method "__init__"',
            f'"model_client" is a required argument, but was not provided',
            f"when the python handler type is used and models are specified in the api spec, "
            f'adding the "model_client" argument is required',
        )

    if getattr(impl, "load_model", None) is None:
        raise UserException(
            f"class {target_class_name}",
            f"required method `load_model` is not defined",
            f"when the python handler type is used and models are specified in the api spec, "
            f"adding the `load_model` method is required",
        )


def are_models_specified(api_spec: Dict) -> bool:
    """
    Checks if models have been specified in the API spec (cortex.yaml).

    Args:
        api_spec: API configuration.
    """
    handler_type = handler_type_from_api_spec(api_spec)

    if handler_type == PythonHandlerType and api_spec["handler"]["multi_model_reloading"]:
        models = api_spec["handler"]["multi_model_reloading"]
    elif handler_type != PythonHandlerType:
        models = api_spec["handler"]["models"]
    else:
        return False

    return models is not None


def is_grpc_enabled(api_spec: Dict) -> bool:
    """
    Checks if the API has the grpc protocol enabled (cortex.yaml).

    Args:
        api_spec: API configuration.
    """
    return api_spec["handler"]["protobuf_path"] is not None


def validate_handler_with_grpc(impl, api_spec: Dict, rpc_method_names: List[str]):
    if not is_grpc_enabled(api_spec):
        return

    target_class_name = impl.__name__
    constructor = getattr(impl, "__init__")
    constructor_arg_spec = inspect.getfullargspec(constructor)
    if "proto_module_pb2" not in constructor_arg_spec.args:
        raise UserException(
            f"class {target_class_name}",
            f"invalid signature for method `__init__`",
            f'"proto_module_pb2" is a required argument, but was not provided',
            f"when a protobuf is specified in the api spec, then that means the grpc protocol is enabled, "
            f'which means that adding the "proto_module_pb2" argument is required',
        )

    for rpc_method_name in rpc_method_names:
        if not util.has_method(impl, rpc_method_name):
            raise UserException(
                f"method {rpc_method_name} hasn't been defined in the Handler class; define one called {rpc_method_name} to match the RPC method from the protobuf definition"
            )

        rpc_handler = getattr(impl, rpc_method_name)
        arg_spec = inspect.getfullargspec(rpc_handler).args
        disallowed_params = list(set(arg_spec).difference(set(["self", "payload", "context"])))
        if len(disallowed_params) > 0:
            raise UserException(
                f"class {target_class_name}",
                f'invalid signature for method "{rpc_method_name}"',
                f'{util.string_plural_with_s("argument", len(disallowed_params))} {util.and_list_with_quotes(disallowed_params)} cannot be used when the grpc protocol is enabled',
            )
