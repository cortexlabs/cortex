# Realtime

Realtime APIs respond to requests synchronously and autoscale based on in-flight request volumes.

Realtime APIs are a good fit for users who want to run stateless containers as a scalable microservice (for example, deploying machine learning models as APIs).

**Key features**

* respond to requests synchronously
* autoscale based on request volume
* avoid cold starts
* scale to zero
* perform rolling updates
* automatically recover from failures and spot instance termination
* perform A/B tests and canary deployments

## How it works

When you deploy a Realtime API, Cortex initializes a pool of worker pods and attaches a proxy sidecar to each of the pods.

The proxy is responsible for receiving incoming requests, queueing them (if necessary), and forwarding them to your pod when it is ready. Autoscaling is based on aggregate in-flight request volume, which is published by the proxy sidecars.

![](https://user-images.githubusercontent.com/808475/146854245-ed0fc153-d083-47d8-a7e2-ac5beb114ee6.png)
