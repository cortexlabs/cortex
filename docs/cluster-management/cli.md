# CLI commands

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## deploy

```text
create or update apis

Usage:
  cortex deploy [CONFIG_FILE] [flags]

Flags:
  -f, --force            override the in-progress api update
  -h, --help             help for deploy
  -p, --profile string   profile (default "local")
  -y, --yes              skip prompts
```

## get

```text
get information about apis

Usage:
  cortex get [API_NAME] [flags]

Flags:
  -h, --help             help for get
  -p, --profile string   profile (default "local")
  -w, --watch            re-run the command every second
```

## logs

```text
stream logs from an api

Usage:
  cortex logs API_NAME [flags]

Flags:
  -h, --help             help for logs
  -p, --profile string   profile (default "local")
```

## refresh

```text
restart all replicas for an api (witout downtime)

Usage:
  cortex refresh API_NAME [flags]

Flags:
  -f, --force            override the in-progress api update
  -h, --help             help for refresh
  -p, --profile string   profile (default "local")
```

## predict

```text
make a prediction request using a json file

Usage:
  cortex predict API_NAME JSON_FILE [flags]

Flags:
      --debug            predict with debug mode
  -h, --help             help for predict
  -p, --profile string   profile (default "local")
```

## delete

```text
delete an api

Usage:
  cortex delete API_NAME [flags]

Flags:
  -f, --force            delete the api without confirmation
  -h, --help             help for delete
  -c, --keep-cache       keep cached data for the api
  -p, --profile string   profile (default "local")
```

## cluster up

```text
spin up a cluster

Usage:
  cortex cluster up [flags]

Flags:
  -c, --config string    path to a cluster configuration file
  -h, --help             help for up
  -p, --profile string   profile (default "aws")
```

## cluster info

```text
get information about a cluster

Usage:
  cortex cluster info [flags]

Flags:
  -c, --config string    path to a cluster configuration file
  -d, --debug            save the current cluster state to a file
  -h, --help             help for info
  -p, --profile string   profile (default "aws")

```

## cluster update

```text
update a cluster

Usage:
  cortex cluster update [flags]

Flags:
  -c, --config string    path to a cluster configuration file
  -h, --help             help for update
  -p, --profile string   profile (default "aws")
```

## cluster down

```text
spin down a cluster

Usage:
  cortex cluster down [flags]

Flags:
  -c, --config string   path to a cluster configuration file
  -h, --help            help for down
```

## version

```text
print the cli and cluster versions

Usage:
  cortex version [flags]

Flags:
  -h, --help             help for version
  -p, --profile string   profile (default "local")
```

## configure

```text
configure a cli profile

Usage:
  cortex configure [--profile=PROFIE_NAME] [flags]

Flags:
  -k, --aws-access-key-id string       set the aws access key id without prompting
  -s, --aws-secret-access-key string   set the aws secret access key without prompting
  -h, --help                           help for configure
  -o, --operator-endpoint string       set the operator endpoint without prompting
  -p, --profile string                 profile (default "local")
  -v, --provider string                set the provider without prompting
```

## configure list

```text
list all configured profiles

Usage:
  cortex configure list [flags]

Flags:
  -h, --help   help for list
```

## configure remove

```text
remove a configured profile

Usage:
  cortex configure remove PROFILE_NAME [flags]

Flags:
  -h, --help   help for remove
```

## completion

```text
generate bash completion scripts

add this to your bashrc or bash profile:
  source <(cortex completion)
or run:
  echo 'source <(cortex completion)' >> ~/.bash_profile  # mac
  echo 'source <(cortex completion)' >> ~/.bashrc  # linux

this will also add the "cx" alias (note: cli completion requires the bash_completion package to be installed on your system)

Usage:
  cortex completion [flags]

Flags:
  -h, --help   help for completion
```
