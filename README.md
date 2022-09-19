# TriggerMesh CLI
Local environment edition.

Project status: Work in progress, initial testing stage.

Working name is `tmcli`.

## Available commands and scenarios

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

Create trigger:

```
tmcli create trigger bar --eventType com.amazon.sqs.message
```

Create target:

```
tmcli create target cloudevents --endpoint https://sockeye-tzununbekov.dev.triggermesh.io --trigger bar 
```

Open sockeye [web-interface](https://sockeye-tzununbekov.dev.triggermesh.io), send the message to SQS queue specified in the source creation step and observe the received CloudEvent in the sockeye tab.

Stop event flow:

```
tmcli stop foo
```

Start event flow:

```
tmcli start foo
```

Print Kubernetes manifest (not applicable at the moment):

```
tmcli dump foo
```
