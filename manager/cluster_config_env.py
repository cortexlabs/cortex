#! /usr/bin/env python

import sys
import yaml

cluster_conifg_path = sys.argv[1]

with open(cluster_conifg_path, "r") as cluster_conifg_file:
    cluster_conifg = yaml.safe_load(cluster_conifg_file)

for key, value in cluster_conifg.items():
    print("export CORTEX_{}={}".format(key.upper(), value))
