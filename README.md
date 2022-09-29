# TriggerMesh CLI
Local environment edition.

Project status: Work in progress, initial testing stage.

Working name is `tmcli`.

## Available commands and scenarios

Commands without the context:

```
tmcli config *
tmcli list
tmcli create broker <broker>
```

Commands with optional context:

```
tmcli dump [broker]
tmcli describe [broker]
tmcli start [broker]
tmcli stop [broker]
tmcli watch [broker]
```

Commands with context from config:

```
tmcli create source *
tmcli create target *
tmcli create trigger *
tmcli create transformation *
```

### Installation

Checkout the code:

```
git clone git@github.com:triggermesh/tmcli.git
```

Install binary:

```
cd tmcli
go install
```

### Local event flow

Create broker:

```
tmcli create broker foo
```

Create source:

```
tmcli create source awssqs --arn <arn> --auth.credentials.accessKeyID=<access key> --auth.credentials.secretAccessKey=<secret key>
```

Watch incoming events:

```
tmcli watch
```

Create target and trigger:

```
tmcli create target cloudevents --endpoint https://sockeye-tzununbekov.dev.triggermesh.io
tmcli create trigger --source foo-awssqssource --target foo-cloudeventstarget
```

Or, in one command:

```
tmcli create target cloudevents --endpoint https://sockeye-tzununbekov.dev.triggermesh.io --source foo-awssqssource
```

Open sockeye [web-interface](https://sockeye-tzununbekov.dev.triggermesh.io), send the message to SQS queue specified in the source creation step and observe the received CloudEvent in the sockeye tab.

Stop event flow:

```
tmcli stop
```

Start event flow:

```
tmcli start
```

Print Kubernetes manifest (not applicable at the moment):

```
tmcli dump
```

Describe the integration:

```
tmcli describe
```

List existing brokers:

```
tmcli list
```
