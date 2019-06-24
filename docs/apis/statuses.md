# Resource Statuses

## Statuses

| Status               | Meaning |
|----------------------|---|
| ready                | API is deployed and ready to serve prediction requests |
| pending              | API is waiting for another resource to be ready, or is initializing |
| updating             | API is performing a rolling update |
| update pending       | API will be updated when the new model is ready; a previous version of this API is ready |
| stopping             | API is stopping |
| stopped              | API is stopped |
| error                | API was not created due to an error; run `cortex logs -v <name>` to view the logs |
| skipped              | API was not created due to an error in another resource |
| update skipped       | API was not updated due to an error in another resource; a previous version of this API is ready |
| upstream error       | API was not created due to an error in one of its dependencies; a previous version of this API may be ready |
| upstream termination | API was not created because one of its dependencies was terminated; a previous version of this API may be ready |
| compute unavailable  | API could not start due to insufficient memory, CPU, or GPU in the cluster; some replicas may be ready |
