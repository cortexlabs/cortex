# Statuses

| Status                   | Meaning |
| :--- | :--- |
| running                  | Task is running |
| succeeded                | Task has finished without errors |
| worker error             | The task has experienced an irrecoverable error, causing the job to fail; check job logs for more details |
| out of memory            | The task has ran out of memory, causing the job to fail; check job logs for more details |
| timed out                | Job was terminated after the specified timeout has elapsed |
| stopped                  | Job was stopped by the user or the Task API was deleted |
