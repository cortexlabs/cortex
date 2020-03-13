# Using GPUs

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

To use GPUs:

1. Make sure your AWS account is subscribed to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM).
2. You may need to [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.
3. Set instance type to an AWS GPU instance (e.g. g4dn.xlarge) when installing Cortex.
4. Set the `gpu` field in the `compute` configuration for your API. One unit of GPU corresponds to one virtual GPU. Fractional requests are not allowed.
