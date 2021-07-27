# Request statuses

| Status            | Meaning                                                               |
| :---              | :---                                                                  |
| in_queue          | Workload is in the queue and is yet to be consumed by the API         |
| in_progress       | Workload has been pulled by the API and is currently being processed  |
| completed         | Workload has completed with success                                   |
| failed            | Workload encountered an error during processing                       |

# Replica states

The replica states of an API can be inspected by running the `cortex describe <api-name>` command. When run, a table is presented that shows how many replicas of the said API are found in each of the following states:

| State | Meaning |
|:---|:---|
| Ready | Replica is running and it has passed the readiness checks |
| ReadyOutOfDate | Replica is running and it has passed the readiness checks (for an out-of-date replica) |
| NotReady | Replica is running but it's not passing the readiness checks; make sure the server is listening on the designed port of the API |
| Requested | Requested number of replicas for a given API |
| Pending | Replica is in a pending state (waiting to get scheduled onto a node) |
| Creating | Replica is in the process of having its containers created |
| ErrImagePull | Replica was not created because one of the specified Docker images was inaccessible at runtime; check that your API's docker images exist and are accessible via your cluster's AWS credentials |
| Failed | Replica couldn't start due to an error; run `cortex logs <name>` to view the logs |
| Killed | Replica has had one of its containers' process(es) killed |
| KilledOOM | Replica was terminated due to excessive memory usage; try allocating more memory to the API and re-deploy |
| Stalled | Replica has been in a pending state for more than 15 minutes; causes like insufficient memory, CPU, GPU or Inf could be culprit; could also be that the node selector on the API is out-of-date |
| Terminating | Replica is currently in the process of being terminated |
| Unknown | Replica is in an unknown state |
