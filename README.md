# TriggerMesh CLI
Local environment edition.

Project status: Work in progress, initial testing stage.

Working name is `tmctl`.

## Available commands and scenarios

Commands without the context:

```
tmctl config *
tmctl list
tmctl create broker <broker>
```

Commands with optional context:

```
tmctl dump [broker]
tmctl describe [broker]
tmctl delete [--broker <broker>] <component>
tmctl start [broker]
tmctl stop [broker]
tmctl watch [broker]
```

Commands with context from config:

```
tmctl create source *
tmctl create target *
tmctl create trigger *
tmctl create transformation *
```

### Installation

Checkout the code:

```
git clone git@github.com:triggermesh/tmctl.git
```

Install binary:

```
cd tmctl
go install
```

### Autocompletion

The CLI can generate completion scripts that can be loaded into the shell
to help use the CLI more easily:

for Bash:
```
source <(tmctl completion bash)
```
or for ZSH:

```
source <(tmctl completion zsh)
```

To make autocompletion load automatically, put this command in one of the
shell profile configuration, e.g.:

```
echo 'source <(tmctl completion bash)' >>~/.bash_profile
```

`tmctl` binary must be available in the `$PATH` to generate and use completion.


### Local event flow

Create broker:

```
tmctl create broker foo
```

Create source:

```
tmctl create source awssqs --arn <arn> --auth.credentials.accessKeyID=<access key> --auth.credentials.secretAccessKey=<secret key>
```

Watch incoming events:

```
tmctl watch
```

Create transformation:
```
tmctl create transformation --sources foo-awssqssource
```

Create target and trigger:

```
tmctl create target cloudevents --endpoint https://sockeye-tzununbekov.dev.triggermesh.io
tmctl create trigger --sources foo-transformation --target foo-cloudeventstarget
```

Or, in one command:

```
tmctl create target cloudevents --endpoint https://sockeye-tzununbekov.dev.triggermesh.io --sources foo-transformation
```

Open sockeye [web-interface](https://sockeye-tzununbekov.dev.triggermesh.io), send the message to SQS queue specified in the source creation step and observe the received CloudEvent in the sockeye tab.

Or send test event manually:

```
tmctl send-event --eventType com.amazon.sqs.message '{"hello":"world"}'
```

Stop event flow:

```
tmctl stop
```

Start event flow:

```
tmctl start
```

Print Kubernetes manifest (not applicable at the moment):

```
tmctl dump
```

Describe the integration:

```
tmctl describe
```

List existing brokers:

```
tmctl list
```

## Contributing

We are happy to review and accept pull requests.

## Commercial Support

TriggerMesh Inc. offers commercial support for the TriggerMesh platform. Email us at <info@triggermesh.com> to get more details.

## License

This software is licensed under the [Apache License, Version 2.0][asl2].

[asl2]: https://www.apache.org/licenses/LICENSE-2.0
