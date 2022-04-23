# Copyright 2022 Cortex Labs, Inc.
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

import sys
import yaml
import os
import pathlib
from jinja2 import Environment, FileSystemLoader

# python render_template.py [CLUSTER_CONFIG_PATH] TEMPLATE_PATH
if __name__ == "__main__":
    if len(sys.argv) == 3:
        yaml_file_path = sys.argv[1]
        template_path = pathlib.Path(sys.argv[2])
    elif len(sys.argv) == 2:
        yaml_file_path = None
        template_path = pathlib.Path(sys.argv[1])
    else:
        raise RuntimeError(f"incorrect number of parameters ({len(sys.argv)})")

    file_loader = FileSystemLoader(str(template_path.parent))
    env = Environment(loader=file_loader)
    env.trim_blocks = True
    env.lstrip_blocks = True
    env.rstrip_blocks = True

    template = env.get_template(str(template_path.name))

    if yaml_file_path:
        with open(yaml_file_path, "r") as f:
            yaml_data = yaml.safe_load(f)
            print(template.render(config=yaml_data, env=os.environ))
    else:
        print(template.render(env=os.environ))
