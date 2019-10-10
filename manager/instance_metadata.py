import boto3
import requests
import argparse
import re
import os
import pathlib
import json

PRICING_ENDPOINT_TEMPLATE = (
    "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/{}/index.json"
)


def download_metadata(args):
    response = requests.get(PRICING_ENDPOINT_TEMPLATE.format(args.region))
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
    with open(args.cache_dir, "w") as outfile:
        json.dump(instance_mapping, outfile)

    return instance_mapping


def get_metadata(args):
    if pathlib.Path(args.cache_dir).exists():
        return json.load(open(args.cache_dir))
    else:
        return download_metadata(args)


units = {"mem": "Mi"}


def set_ec2_metadata(args):
    instance_mapping = get_metadata(args)
    instance_type = instance_mapping[args.instance_type]
    print("{}{}".format(instance_type.get(args.feature, "0"), units.get(args.feature, "")))


def main():
    parser = argparse.ArgumentParser()
    na = parser.add_argument_group("required named arguments")
    na.add_argument("--region", required=True, help="AWS Region")
    na.add_argument("--instance-type", required=True, help="Instance type")
    na.add_argument("--feature", required=True, help="Feature to get")
    na.add_argument("--cache-dir", required=True, help="Cache dir")
    args = parser.parse_args()
    set_ec2_metadata(args)


if __name__ == "__main__":
    main()
