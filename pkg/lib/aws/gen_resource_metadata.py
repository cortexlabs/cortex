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

import requests
import re
from string import Template

# https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/
REGIONS = [
    "us-east-1",  # N. Virginia
    "us-east-2",  # Ohio
    "us-west-2",  # Oregon
    "ca-central-1",  # Montreal
    "sa-east-1",  # Sao Paulo
    "eu-west-1",  # Ireland
    "eu-central-1",  # Frankfurt
    "eu-west-2",  # London
    "eu-west-3",  # Paris
    "eu-north-1",  # Stockholm
    "me-south-1",  # Bahrain
    "ap-southeast-1",  # Singapore
    "ap-northeast-1",  # Tokyo
    "ap-southeast-2",  # Sydney
    "ap-northeast-2",  # Seoul
    "ap-south-1",  # Mumbai
    "ap-east-1",  # Hong Kong
]

OUTPUT_FILE_NAME = "resource_metadata.go"

EC2_PRICING_ENDPOINT_TEMPLATE = (
    "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/{}/index.json"
)

EKS_PRICING_ENDPOINT_TEMPLATE = (
    "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEKS/current/{}/index.json"
)


def get_instance_metadatas(pricing):
    instance_mapping = {}

    for product_id, product in pricing["products"].items():
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
        price_dimensions = list(pricing["terms"]["OnDemand"][product["sku"]].values())[0][
            "priceDimensions"
        ]

        price = list(price_dimensions.values())[0]["pricePerUnit"]["USD"]

        instance_type = product["attributes"]["instanceType"]
        metadata = {
            "cpu": int(product["attributes"]["vcpu"]),
            "mem": int(
                float(re.sub("[^0-9\\.]", "", product["attributes"]["memory"].split(" ")[0])) * 1024
            ),
            "price": float(price),
            "gpu": 0,
        }
        if product["attributes"].get("gpu") is not None:
            metadata["gpu"] = product["attributes"]["gpu"]
        instance_mapping[instance_type] = metadata

    return instance_mapping


def get_elb_metadata(pricing):
    for product_id, product in pricing["products"].items():
        if product.get("attributes") is None:
            continue
        if product.get("productFamily") != "Load Balancer":
            continue
        if product["attributes"].get("group") != "ELB:Balancer":
            continue
        if product["attributes"].get("operation") != "LoadBalancing":
            continue

        price_dimensions = list(pricing["terms"]["OnDemand"][product["sku"]].values())[0][
            "priceDimensions"
        ]
        price = list(price_dimensions.values())[0]["pricePerUnit"]["USD"]
        return {"price": float(price)}


def get_nat_metadata(pricing):
    for product_id, product in pricing["products"].items():
        if product.get("attributes") is None:
            continue
        if product.get("productFamily") != "NAT Gateway":
            continue
        if product["attributes"].get("group") != "NGW:NatGateway":
            continue
        if product["attributes"].get("operation") != "NatGateway":
            continue
        if not product["attributes"].get("usagetype", "").endswith("-Hours"):
            continue

        price_dimensions = list(pricing["terms"]["OnDemand"][product["sku"]].values())[0][
            "priceDimensions"
        ]
        price = list(price_dimensions.values())[0]["pricePerUnit"]["USD"]
        return {"price": float(price)}


def get_ebs_metadata(pricing):
    for product_id, product in pricing["products"].items():
        if product.get("attributes") is None:
            continue
        if product.get("productFamily") != "Storage":
            continue
        if product["attributes"].get("volumeApiName") != "gp2":
            continue

        price_dimensions = list(pricing["terms"]["OnDemand"][product["sku"]].values())[0][
            "priceDimensions"
        ]
        price = list(price_dimensions.values())[0]["pricePerUnit"]["USD"]
        return {"price": float(price)}


def get_eks_price(region):
    response = requests.get(EKS_PRICING_ENDPOINT_TEMPLATE.format(region))
    pricing = response.json()

    for product_id, product in pricing["products"].items():
        if product.get("attributes") is None:
            continue
        if product.get("productFamily") != "Compute":
            continue
        if product["attributes"].get("servicecode") != "AmazonEKS":
            continue
        if product["attributes"].get("operation") != "CreateOperation":
            continue
        if not product["attributes"].get("usagetype", "").endswith("-AmazonEKS-Hours:perCluster"):
            continue

        price_dimensions = list(pricing["terms"]["OnDemand"][product["sku"]].values())[0][
            "priceDimensions"
        ]
        price = list(price_dimensions.values())[0]["pricePerUnit"]["USD"]
        return float(price)


file_template = Template(
    """/*
Copyright 2020 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was generated by go generate; DO NOT EDIT

package aws

import (
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

type InstanceMetadata struct {
	Region string             `json:"region"`
	Type   string             `json:"type"`
	Memory kresource.Quantity `json:"memory"`
	CPU    kresource.Quantity `json:"cpu"`
	GPU    int64              `json:"gpu"`
	Price  float64            `json:"price"`
}

type ELBMetadata struct {
	Region string  `json:"region"`
	Price  float64 `json:"price"`
}

type NATMetadata struct {
	Region string  `json:"region"`
	Price  float64 `json:"price"`
}

type EBSMetadata struct {
	Region string  `json:"region"`
	Price  float64 `json:"price"`
}

// region -> instance type -> instance metadata
var InstanceMetadatas = map[string]map[string]InstanceMetadata{
    ${instance_region_map}
}

// region -> ELB metadata
var ELBMetadatas = map[string]ELBMetadata{
    ${elb_region_map}
}

// region -> NAT metadata
var NATMetadatas = map[string]NATMetadata{
    ${nat_region_map}
}

// region -> EBS metadata
var EBSMetadatas = map[string]EBSMetadata{
    ${ebs_region_map}
}

// region -> EKS price
var EKSPrices = map[string]float64{
    ${eks_region_map}
}
"""
)

instance_region_map_template = Template(
    """"${region}": map[string]InstanceMetadata{
	${instance_metadatas}
},
"""
)

instance_metadata_template = Template(
    """"${type}": {Region: "${region}", Type: "${type}", Memory: kresource.MustParse("${memory}Mi"), CPU: kresource.MustParse("${cpu}"), GPU: ${gpu}, Price: ${price}},
"""
)

elb_region_map_template = Template(
    """"${region}": {Region: "${region}", Price: ${price}},
"""
)

nat_region_map_template = Template(
    """"${region}": {Region: "${region}", Price: ${price}},
"""
)

ebs_region_map_template = Template(
    """"${region}": {Region: "${region}", Price: ${price}},
"""
)

eks_region_map_template = Template(
    """"${region}": ${price},
"""
)


def main():
    instance_region_map_str = ""
    elb_region_map_str = ""
    nat_region_map_str = ""
    ebs_region_map_str = ""
    eks_region_map_str = ""

    for i, region in enumerate(sorted(REGIONS), start=1):
        print("generating region {}/{} ({})...".format(i, len(REGIONS), region))

        response = requests.get(EC2_PRICING_ENDPOINT_TEMPLATE.format(region))
        pricing = response.json()

        instance_metadatas = get_instance_metadatas(pricing)
        elb_metadata = get_elb_metadata(pricing)
        nat_metadata = get_nat_metadata(pricing)
        ebs_metadata = get_ebs_metadata(pricing)
        eks_price = get_eks_price(region)

        instance_metadatas_str = ""

        for instance_type in sorted(instance_metadatas.keys()):
            metadata = instance_metadatas[instance_type]
            instance_metadatas_str += instance_metadata_template.substitute(
                {
                    "region": region,
                    "type": instance_type,
                    "memory": metadata["mem"],
                    "cpu": metadata["cpu"],
                    "gpu": metadata["gpu"],
                    "price": metadata["price"],
                }
            )

        instance_region_map_str += instance_region_map_template.substitute(
            {"region": region, "instance_metadatas": instance_metadatas_str}
        )
        elb_region_map_str += elb_region_map_template.substitute(
            {"region": region, "price": elb_metadata["price"]}
        )
        nat_region_map_str += nat_region_map_template.substitute(
            {"region": region, "price": nat_metadata["price"]}
        )
        ebs_region_map_str += ebs_region_map_template.substitute(
            {"region": region, "price": ebs_metadata["price"]}
        )
        eks_region_map_str += eks_region_map_template.substitute(
            {"region": region, "price": eks_price}
        )

    file_str = file_template.substitute(
        {
            "instance_region_map": instance_region_map_str,
            "elb_region_map": elb_region_map_str,
            "nat_region_map": nat_region_map_str,
            "ebs_region_map": ebs_region_map_str,
            "eks_region_map": eks_region_map_str,
        }
    )

    with open(OUTPUT_FILE_NAME, "w") as f:
        print("writing {}...".format(OUTPUT_FILE_NAME))
        f.write(file_str)
        print("✓ done")


if __name__ == "__main__":
    main()
