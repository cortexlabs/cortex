# Statuses

| Status                   | Meaning |
| :--- | :--- |
| enqueuing                | Job is being split into batches and placed into a queue |
| running                  | Workers are retrieving batches from the queue and running inference |
| succeeded                | Workers completed all items in the queue without any failures |
| failed while enqueuing   | Failure occurred while enqueuing; check job logs for more details |
| completed with failures  | Workers completed all items in the queue but some of the batches weren't processed successfully and raised exceptions; check job logs for more details |
| worker error             | One or more workers experienced an irrecoverable error, causing the job to fail; check job logs for more details |
| out of memory            | One or more workers ran out of memory, causing the job to fail; check job logs for more details |
| timed out                | Job was terminated after the specified timeout has elapsed |
| stopped                  | Job was stopped by the user or the Batch API was deleted |
