# Copyright 2019 Cortex Labs, Inc.
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

import requests
import sys
import re
import os
import pathlib
import json
import yaml

PRICING_ENDPOINT_TEMPLATE = (
    "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/{}/index.json"
)


def download_metadata(cluster_config):
    response = requests.get(PRICING_ENDPOINT_TEMPLATE.format(cluster_config["region"]))
    offers = response.json()

    instance_mapping = {}

    for product_id, product in offers["products"].items():
        if product.get("attributes") is None:
            continue
        if product["attributes"].get("servicecode") != "AmazonEC2":
            continue
        if product["attributes"].get("tenancy") != "Shared":
            continue
        if product["attributes"].get("operatingSystem") != "Linux":
            continue
        if product["attributes"].get("capacitystatus") != "Used":
            continue
        if product["attributes"].get("operation") != "RunInstances":
            continue
        price_dimensions = list(offers["terms"]["OnDemand"][product["sku"]].values())[0][
            "priceDimensions"
        ]

        price = list(price_dimensions.values())[0]["pricePerUnit"]["USD"]

        instance_type = product["attributes"]["instanceType"]
        metadata = {
            "sku": product["sku"],
            "instance_type": instance_type,
            "cpu": int(product["attributes"]["vcpu"]),
            "mem": int(
                float(re.sub("[^0-9\\.]", "", product["attributes"]["memory"].split(" ")[0])) * 1024
            ),
            "price": float(price),
        }
        if product["attributes"].get("gpu") is not None:
            metadata["gpu"] = product["attributes"]["gpu"]
        instance_mapping[instance_type] = metadata

    return instance_mapping


def set_ec2_metadata(cluster_config_path, internal_cluster_config_path):
    with open(cluster_config_path, "r") as f:
        cluster_config = yaml.safe_load(f)
    instance_mapping = download_metadata(cluster_config)
    instance_metadata = instance_mapping[cluster_config["instance_type"]]

    internal_cluster_config = {
        "instance_mem": str(instance_metadata["mem"]) + "Mi",
        "instance_cpu": str(instance_metadata["cpu"]),
        "instance_gpu": int(instance_metadata.get("gpu", 0)),
    }

    with open(internal_cluster_config_path, "w") as f:
        yaml.dump(internal_cluster_config, f)


def main():
    set_ec2_metadata(sys.argv[1], sys.argv[2])


if __name__ == "__main__":
    main()
