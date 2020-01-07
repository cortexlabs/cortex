import sys
import yaml
import os
from copy import deepcopy
import pathlib
from jinja2 import Environment, FileSystemLoader

if __name__ == "__main__":
    configmap_yaml_path = sys.argv[1]
    template_path = pathlib.Path(sys.argv[2])

    file_loader = FileSystemLoader(str(template_path.parent))
    env = Environment(loader=file_loader)
    env.trim_blocks = True
    env.lstrip_blocks = True
    env.rstrip_blocks = True

    template = env.get_template(str(template_path.name))
    with open(sys.argv[1], "r") as f:
        cluster_configmap = yaml.safe_load(f)
        print(template.render(config=cluster_configmap))
