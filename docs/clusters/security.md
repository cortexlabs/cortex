# Security

## Private cluster subnets

By default, instances are created in public subnets and are assigned public IP addresses. You can configure all instances in your cluster to use private subnets by setting `subnet_visibility: private` in your [cluster configuration](install.md) file before creating your cluster. If private subnets are used, instances will not have public IP addresses, and Cortex will create a NAT gateway to allow outgoing network requests.

## Private APIs

See [networking](networking/index.md) for a discussion of API visibility.

## Private operator

By default, the Cortex cluster operator's load balancer is internet-facing, and therefore publicly accessible (the operator is what the `cortex` CLI connects to). The operator's load balancer can be configured to be private by setting `operator_load_balancer_scheme: internal` in your [cluster configuration](install.md) file. If you do this, you will need to configure [VPC Peering](networking/vpc-peering.md) to allow your CLI to connect to the Cortex operator (this will be necessary to run any `cortex` commands).
