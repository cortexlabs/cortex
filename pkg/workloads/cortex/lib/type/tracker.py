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


class Tracker:
    def __init__(self, **kwargs):
        self.key = kwargs.get("key")
        self.model_type = kwargs["model_type"]

    def extract_predicted_value(self, prediction):
        if self.key is not None:
            if type(prediction) != dict:
                raise ValueError(
                    "failed to track key '{}': expected prediction to be of type dict but found '{}'".format(
                        self.key, type(prediction)
                    )
                )
            if prediction.get(self.key) is None:
                raise ValueError(
                    "failed to track key '{}': not found in prediction".format(self.key)
                )
            predicted_value = prediction[self.key]
        else:
            predicted_value = prediction

        if self.model_type == "classification":
            if type(predicted_value) != str and type(predicted_value) != int:
                raise ValueError(
                    "failed to track classification prediction: expected type 'str' or 'int' but encountered '{}'".format(
                        type(predicted_value)
                    )
                )
            return str(predicted_value)
        else:
            if type(predicted_value) != float and type(predicted_value) != int:
                raise ValueError(
                    "failed to track regression prediction: expected type 'float' or 'int' but encountered '{}'".format(
                        type(predicted_value)
                    )
                )
        return predicted_value
