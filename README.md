```
#HAPPY
go run . create broker bob

go run . create source awssqs --arn=arn:aws:sqs:us-west-1:397904378622:test  --auth.credentials.accessKeyID=AKIAVZJHS7L7FYPAUPJ5 --auth.credentials.secretAccessKey=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


go run . create target cloudevents --endpoint http://tmdebugger-personal-org-75-personal-ns-75.k.triggermesh.io

go run . create trigger --sources bob-awssqssource --target bob-cloudeventstarget

go run . dump -o docker-compose
go run . compose
```


Local instance data:
```
go run . create target cloudevents --endpoint http://192.168.1.15:8080
go run . create trigger --eventTypes com.test --target bob-cloudeventstarget
go run . send-event --eventType com.test '{"hello":"world"}'

```

KafkaSource
WebhookSource
CloudEventsSource
AWSSQSSource
AWSKinesisSource
AzureEventHubSource
HTTPPollerSource
GoogleCloudPubSubSource


I agree that dumping from k8s to docker-compose & back is cool... but designing from here is not. In my opionon this is useles technology wrapped with the interface it is. We are doing the exact same thing as we tried to do with tmcli and it failed. So i dont understand at all what we are trying to acomplish here. 

This is maybe, an "oh hey, thats cool" repo. NOT a "I need this every day of my life to do my job or it sucks" repo.. 

No one asked for this, and everyone we have brought it up to (that i was in the prensense of) dismissed, rejected, or said "thats cool, but I probably would not use this"-Noah Kreiger



```
go run . create trigger --sources awssqs --target foo-cloudeventstarget


#BROKEN
go run .  create transformation --sources awssqs

```


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
