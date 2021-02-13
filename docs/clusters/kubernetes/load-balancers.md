## AWS Load-Balancer

With the default helm configuration, you get a AWS Classic Load Balancer.

To get NLBs (Network Load Balancers), you need the following config:

```yaml
# values.yaml

networking:
    # for api ingress (RealtimeAPI, BatchAPI, TaskAPI, TrafficSplitter)
    api-ingress:
        gateways:
            istio-ingressgateway:
                serviceAnnotations:
                    - service.beta.kubernetes.io/aws-load-balancer-type=nlb
                    - service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled=true
                    - service.beta.kubernetes.io/aws-load-balancer-backend-protocol=tcp
    # for operator ingress (any command done through the CLI/Python Client)
    operator-ingress:
        gateways:
            istio-ingressgateway:
                serviceAnnotations:
                    - service.beta.kubernetes.io/aws-load-balancer-type=nlb
                    - service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled=true
                    - service.beta.kubernetes.io/aws-load-balancer-backend-protocol=tcp
```

To get private load balancers, you need the following config:

```yaml
# values.yaml

networking:
    # for api ingress (RealtimeAPI, BatchAPI, TaskAPI, TrafficSplitter)
    api-ingress:
        gateways:
            istio-ingressgateway:
                serviceAnnotations:
                    - service.beta.kubernetes.io/aws-load-balancer-internal=true
    # for operator ingress (any command done through the CLI/Python Client)
    operator-ingress:
        gateways:
            istio-ingressgateway:
                serviceAnnotations:
                    - service.beta.kubernetes.io/aws-load-balancer-internal=true
```

*Note: if you make your operator load balancer private, you must be within the cluster's VPC or configure VPC Peering to connect your CLI to your cluster operator.*

## GCP Load-Balancer

To configure your load balancers to be private, you need the following config:

```yaml
# values.yaml

networking:
    # for api ingress (RealtimeAPI, BatchAPI, TaskAPI, TrafficSplitter)
    api-ingress:
        gateways:
            istio-ingressgateway:
                serviceAnnotations:
                    - cloud.google.com/load-balancer-type=Internal
    # for operator ingress (any command done through the CLI/Python Client)
    operator-ingress:
        gateways:
            istio-ingressgateway:
                serviceAnnotations:
                    - cloud.google.com/load-balancer-type=Internal
```

*Note: if you make your operator load balancer private, you must be within the cluster's VPC or configure VPC Peering to connect your CLI to your cluster operator.*
