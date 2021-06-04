# Realtime APIs

Realtime APIs respond to requests synchronously and autoscale based on in-flight request volumes.

RealtimeAPI is a good fit for users who want to run stateless containers as a scalable microservice. A common use cases for the RealtimeAPI is to deploy models as APIs.

## How it works

When you deploy a RealtimeAPI, Cortex initializes a pool of worker pods and attaches a proxy side car to each of the pods.

The proxy is responsible for queuing incoming requests and forwarding them to your pod when it is ready. Autoscaling decisions are made based on request volume metrics published by the proxy sidecars.

![]()