/*
Copyright 2021 Cortex Labs, Inc.

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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

// copied from https://github.com/weaveworks/eksctl/blob/c211e68d3c8cf3c7f800768bfa0251dda17e011c/pkg/apis/eksctl.io/v1alpha5/types.go
// most of this code can be removed once eksctl can be imported: https://github.com/weaveworks/eksctl/issues/813
const (
	eksResourceAccountStandard = "602401143452"

	// eksResourceAccountAPEast1 defines the AWS EKS account ID that provides node resources in ap-east-1 region
	eksResourceAccountAPEast1 = "800184023465"

	// eksResourceAccountMESouth1 defines the AWS EKS account ID that provides node resources in me-south-1 region
	eksResourceAccountMESouth1 = "558608220178"

	// eksResourceAccountCNNorthWest1 defines the AWS EKS account ID that provides node resources in cn-northwest-1 region
	eksResourceAccountCNNorthWest1 = "961992271922"

	// eksResourceAccountCNNorth1 defines the AWS EKS account ID that provides node resources in cn-north-1
	eksResourceAccountCNNorth1 = "918309763551"

	// eksResourceAccountAFSouth1 defines the AWS EKS account ID that provides node resources in af-south-1
	eksResourceAccountAFSouth1 = "877085696533"

	// eksResourceAccountEUSouth1 defines the AWS EKS account ID that provides node resources in eu-south-1
	eksResourceAccountEUSouth1 = "590381155156"

	// eksResourceAccountUSGovWest1 defines the AWS EKS account ID that provides node resources in us-gov-west-1
	eksResourceAccountUSGovWest1 = "013241004608"

	// eksResourceAccountUSGovEast1 defines the AWS EKS account ID that provides node resources in us-gov-east-1
	eksResourceAccountUSGovEast1 = "151742754352"
)

// Regions
const (
	// RegionUSWest1 represents the US West Region North California
	RegionUSWest1 = "us-west-1"

	// RegionUSWest2 represents the US West Region Oregon
	RegionUSWest2 = "us-west-2"

	// RegionUSEast1 represents the US East Region North Virginia
	RegionUSEast1 = "us-east-1"

	// RegionUSEast2 represents the US East Region Ohio
	RegionUSEast2 = "us-east-2"

	// RegionCACentral1 represents the Canada Central Region
	RegionCACentral1 = "ca-central-1"

	// RegionEUWest1 represents the EU West Region Ireland
	RegionEUWest1 = "eu-west-1"

	// RegionEUWest2 represents the EU West Region London
	RegionEUWest2 = "eu-west-2"

	// RegionEUWest3 represents the EU West Region Paris
	RegionEUWest3 = "eu-west-3"

	// RegionEUNorth1 represents the EU North Region Stockholm
	RegionEUNorth1 = "eu-north-1"

	// RegionEUCentral1 represents the EU Central Region Frankfurt
	RegionEUCentral1 = "eu-central-1"

	// RegionEUSouth1 represents te Eu South Region Milan
	RegionEUSouth1 = "eu-south-1"

	// RegionAPNorthEast1 represents the Asia-Pacific North East Region Tokyo
	RegionAPNorthEast1 = "ap-northeast-1"

	// RegionAPNorthEast2 represents the Asia-Pacific North East Region Seoul
	RegionAPNorthEast2 = "ap-northeast-2"

	// RegionAPNorthEast3 represents the Asia-Pacific North East region Osaka
	RegionAPNorthEast3 = "ap-northeast-3"

	// RegionAPSouthEast1 represents the Asia-Pacific South East Region Singapore
	RegionAPSouthEast1 = "ap-southeast-1"

	// RegionAPSouthEast2 represents the Asia-Pacific South East Region Sydney
	RegionAPSouthEast2 = "ap-southeast-2"

	// RegionAPSouth1 represents the Asia-Pacific South Region Mumbai
	RegionAPSouth1 = "ap-south-1"

	// RegionAPEast1 represents the Asia Pacific Region Hong Kong
	RegionAPEast1 = "ap-east-1"

	// RegionMESouth1 represents the Middle East Region Bahrain
	RegionMESouth1 = "me-south-1"

	// RegionSAEast1 represents the South America Region Sao Paulo
	RegionSAEast1 = "sa-east-1"

	// RegionAFSouth1 represents the Africa Region Cape Town
	RegionAFSouth1 = "af-south-1"

	// RegionCNNorthwest1 represents the China region Ningxia
	RegionCNNorthwest1 = "cn-northwest-1"

	// RegionCNNorth1 represents the China region Beijing
	RegionCNNorth1 = "cn-north-1"

	// RegionUSGovWest1 represents the region GovCloud (US-West)
	RegionUSGovWest1 = "us-gov-west-1"

	// RegionUSGovEast1 represents the region GovCloud (US-East)
	RegionUSGovEast1 = "us-gov-east-1"

	// DefaultRegion defines the default region, where to deploy the EKS cluster
	DefaultRegion = RegionUSWest2
)

// SupportedRegions are the regions where EKS is available
func SupportedRegions() []string {
	return []string{
		RegionUSWest1,
		RegionUSWest2,
		RegionUSEast1,
		RegionUSEast2,
		RegionCACentral1,
		RegionEUWest1,
		RegionEUWest2,
		RegionEUWest3,
		RegionEUNorth1,
		RegionEUCentral1,
		RegionEUSouth1,
		RegionAPNorthEast1,
		RegionAPNorthEast2,
		RegionAPNorthEast3,
		RegionAPSouthEast1,
		RegionAPSouthEast2,
		RegionAPSouth1,
		RegionAPEast1,
		RegionMESouth1,
		RegionSAEast1,
		RegionAFSouth1,
		// RegionCNNorthwest1,
		// RegionCNNorth1,
		// RegionUSGovWest1,
		// RegionUSGovEast1,
	}
}

func EKSResourceAccountID(region string) string {
	switch region {
	case RegionAPEast1:
		return eksResourceAccountAPEast1
	case RegionMESouth1:
		return eksResourceAccountMESouth1
	case RegionCNNorthwest1:
		return eksResourceAccountCNNorthWest1
	case RegionCNNorth1:
		return eksResourceAccountCNNorth1
	case RegionUSGovWest1:
		return eksResourceAccountUSGovWest1
	case RegionUSGovEast1:
		return eksResourceAccountUSGovEast1
	case RegionAFSouth1:
		return eksResourceAccountAFSouth1
	case RegionEUSouth1:
		return eksResourceAccountEUSouth1
	default:
		return eksResourceAccountStandard
	}
}

func main() {
	destFile := mustExtractArg()
	k8sVersion := "1.18"
	regions := map[string]map[string]string{}
	for _, region := range SupportedRegions() {
		fmt.Print(region)
		sess := session.New(&aws.Config{Region: aws.String(region)})
		svc := ec2.New(sess)
		cpuAMI, err := FindImage(svc, EKSResourceAccountID(region), fmt.Sprintf("amazon-eks-node-%s-v*", k8sVersion))
		if err != nil {
			log.Fatal(err.Error())
		}
		acceleratedAMI, err := FindImage(svc, EKSResourceAccountID(region), fmt.Sprintf("amazon-eks-gpu-node-%s-v*", k8sVersion))
		if err != nil {
			log.Fatal(err.Error())
		}
		regions[region] = map[string]string{
			"cpu":         cpuAMI,
			"accelerated": acceleratedAMI,
		}
		fmt.Println(" ✓")
	}

	k8sVersionMap := map[string]interface{}{}
	k8sVersionMap[k8sVersion] = regions
	marshalledBytes, err := json.MarshalIndent(k8sVersionMap, "", "\t")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = ioutil.WriteFile(destFile, marshalledBytes, 0664)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func FindImage(ec2api ec2iface.EC2API, ownerAccount, namePattern string) (string, error) {
	input := &ec2.DescribeImagesInput{
		Owners: []*string{&ownerAccount},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{&namePattern},
			},
			{
				Name:   aws.String("virtualization-type"),
				Values: []*string{aws.String("hvm")},
			},
			{
				Name:   aws.String("root-device-type"),
				Values: []*string{aws.String("ebs")},
			},
			{
				Name:   aws.String("is-public"),
				Values: []*string{aws.String("true")},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	}

	output, err := ec2api.DescribeImages(input)
	if err != nil {
		return "", errors.Wrapf(err, "error querying AWS for images")
	}

	if len(output.Images) < 1 {
		return "", nil
	}

	if len(output.Images) == 1 {
		return *output.Images[0].ImageId, nil
	}

	// Sort images so newest is first
	sort.Slice(output.Images, func(i, j int) bool {
		//nolint:gosec
		creationLeft, _ := time.Parse(time.RFC3339, *output.Images[i].CreationDate)
		//nolint:gosec
		creationRight, _ := time.Parse(time.RFC3339, *output.Images[j].CreationDate)
		return creationLeft.After(creationRight)
	})

	return *output.Images[0].ImageId, nil
}

func mustExtractArg() string {
	if len(os.Args) != 2 {
		fmt.Println("usage: go run generate_ami_mapping.go <abs_dest_path>")
		os.Exit(1)
	}

	return os.Args[1]
}
