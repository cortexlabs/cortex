# Resource Statuses

## Statuses

| Status               | Meaning |
|----------------------|---|
| live                 | API is deployed and ready to serve prediction requests (at least one replica is running) |
| pending              | API is waiting for another resource to be ready |
| creating             | API is being created |
| stopping             | API is stopping |
| stopped              | API is stopped |
| error                | API was not created due to an error; run `cortex logs <name>` to view the logs |
| skipped              | API was not created due to an error in another resource |
| upstream error       | API was not created due to an error in one of its dependencies; a previous version of this API may be ready |
| upstream termination | API was not created because one of its dependencies was terminated; a previous version of this API may be ready |
| compute unavailable  | API could not start due to insufficient memory, CPU, or GPU in the cluster; some replicas may be ready |
