# Resource Statuses

## Statuses

| Status                  | Meaning |
|-------------------------|---|
| ready                   | Resource is ready |
| pending                 | Resource is waiting for another resource to be ready, or its workload is initializing |
| running, ingesting,<br>aggregating, transforming,<br>generating, training | Resource is being created |
| error                   | Resource was not created due to an error; run `cortex logs -v <name>` to view the logs |
| skipped                 | Resource was not created due to an error in another resource in the same workload |
| terminated              | Resource was terminated |
| terminated (out of mem) | Resource was terminated due to insufficient memory |
| upstream error          | Resource was not created due to an error in one of its dependencies |
| upstream termination    | Resource was not created because one of its dependencies was terminated |
| compute unavailable     | Resource's workload could not start due to insufficient memory, CPU, or GPU in the cluster |
