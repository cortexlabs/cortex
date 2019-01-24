# Security

## Cluster access

In order to connect to the cluster, you must provide valid AWS credentials for a user with access to the account. The CLI can be configured using the command `cortex configure`.

## API access

By default, your APIs will be accessible to all traffic. You can restrict access using AWS security groups. Specifically, you will need to edit the security group with the description: "Security group for Kubernetes ELB <ELB name> (cortex/nginx-controller-apis)".
