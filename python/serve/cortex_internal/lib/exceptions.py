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

from collections import deque


class WithBreak(Exception):
    """
    Gracefully exit with clauses.
    """

    pass


class CortexException(Exception):
    def __init__(self, *messages):
        super().__init__(": ".join(messages))
        self.errors = deque(messages)

    def wrap(self, *messages):
        self.errors.extendleft(reversed(messages))

    def __str__(self):
        return self.stringify()

    def __repr__(self):
        return self.stringify()

    def stringify(self):
        return "error: " + ": ".join(self.errors)


class UserException(CortexException):
    def __init__(self, *messages):
        super().__init__(*messages)


class UserRuntimeException(UserException):
    def __init__(self, *messages):
        msg_list = list(messages)
        msg_list.append("runtime exception")
        super().__init__(*msg_list)
