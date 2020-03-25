# Telemetry

By default, Cortex sends anonymous usage data to Cortex Labs.

## What data is collected?

If telemetry is enabled, events and errors are collected. Each time you run a command an event will be sent with a randomly generated unique CLI ID and the name of the command. For example, if you run `cortex deploy`, Cortex Labs will receive an event of the structure {id: 1234, command: "deploy"}. In addition, the operator sends heartbeats that include cluster metrics like the types of instances running in your cluster.

## Why is this data being collected?

Telemetry helps us make Cortex better. For example, we discovered that people are running `cortex delete` more times than we expected and realized that our documentation doesn't explain clearly that `cortex deploy` is declarative and can be run consecutively without deleting APIs.

## How do I opt out?

If you'd like to disable telemetry, modify your `~/.cortex/cli.yaml` file (or create it if it doesn't exist) and add `telemetry: false`.
