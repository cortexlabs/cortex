# Telemetry

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

By default, Cortex sends anonymous usage data to Cortex Labs.

## What data is collected?

If telemetry is enabled, events and errors are collected. Each time you run a command an event will be sent with a randomly generated unique CLI ID and the name of the command. For example, if you run `cortex get`, Cortex Labs will receive an event of the structure `{id: 1234, command: "get"}`. In addition, the operator sends heartbeats that include cluster metrics like the types of instances running in your cluster.

## How do I opt out?

If you'd like to disable telemetry, modify your `~/.cortex/cli.yaml` file (or create it if it doesn't exist) and add `telemetry: false`.
