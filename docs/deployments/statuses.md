# API statuses

| Status                | Meaning |
| :--- | :--- |
| live                  | API is deployed and ready to serve prediction requests (at least one replica is running) |
| updating              | API is updating |
| error                 | API was not created due to an error; run `cortex logs <name>` to view the logs |
| error (out of memory) | API was terminated due to excessive memory usage; try allocating more memory to the API and re-deploying |
| compute unavailable   | API could not start due to insufficient memory, CPU, or GPU in the cluster; some replicas may be ready |
