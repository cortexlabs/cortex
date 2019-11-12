import boto3
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


def get_metadata(cluster_config):
    return download_metadata(cluster_config)


def set_ec2_metadata(cluster_config_path):
    with open(cluster_config_path, "r") as cluster_config_file:
        cluster_config = yaml.safe_load(cluster_config_file)
    instance_mapping = get_metadata(cluster_config)
    instance_type = instance_mapping[cluster_config["instance_type"]]

    cluster_config["instance_mem"] = str(instance_type["mem"]) + "Mi"
    cluster_config["instance_cpu"] = str(instance_type["cpu"])
    cluster_config["instance_gpu"] = int(instance_type.get("gpu", 0))

    with open(cluster_config_path, "w") as cluster_config_file:
        yaml.dump(cluster_config, cluster_config_file, default_flow_style=False)


def main():
    set_ec2_metadata(sys.argv[1])


if __name__ == "__main__":
    main()
