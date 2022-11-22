# TriggerMesh CLI

`tmctl` is the TriggerMesh CLI tool to create, manage and debug event-driven integration apps. The key idea of this CLI is to provide a simple user experience in the local environment and support further migration to Kubernetes.


For the full documentation of the TriggerMesh and its tooling, please visit [docs.triggermesh.io](https://docs.triggermesh.io).

## Requirements

The CLI runs TriggerMesh components locally as containers, therefore Docker engine must be running on the machine where CLI is installed.

## Installation

TriggerMesh CLI can be installed from different sources: brew repository, pre-built binary, or compiled from the sources.

### Brew

```
brew install tmctl
```

### Pre-built binary

For MacOS on Apple Silicon chips:

```
curl -L https://github.com/triggermesh/tmctl/releases/download/v0.0.1/tmctl_0.0.1_darwin_arm64 -o /usr/local/bin/tmctl && chmod +x /usr/local/bin/tmctl
```

Linux, AMD64:

```
curl -L https://github.com/triggermesh/tmctl/releases/download/v0.0.1/tmctl_0.0.1_linux_amd64 -o /usr/local/bin/tmctl && chmod +x /usr/local/bin/tmctl
```

To view more versions and architectures of pre-built binaries please visit our Github [release page](https://github.com/triggermesh/tmctl/releases/latest). 

### Source

`go` compiler of the latest version is recommended to build `tmctl` binary from the source:

```
git clone git@github.com:triggermesh/tmctl.git
cd tmctl
go install
```

### Autocompletion

After `tmctl` is installed and available in system's `$PATH`, command-line [completion](https://en.wikipedia.org/wiki/Command-line_completion) should be configured as it improves the CLI user experience. To configure command-line completion, please use the "help" command output for the shell of your choice, for example:

```
tmctl completion bash --help
``` 
or

```
tmctl completion zsh --help
```

_NOTE: for the Bash shell, `bash-completion` of version *2* is recommended._

## Usage

The CLI commands provide a way to manage TriggerMesh resources locally and deploy them on Kubernetes cluster without the need to write YAML files. All commands support `--help` argument to get the description and usage:

```
$ tmctl help
tmctl is a CLI to help you create event brokers, sources, targets and transformations.

Available Commands:
  brokers     Show the list of brokers
  completion  Generate the autocompletion script for the specified shell
  config      Read and write config values
  create      Create TriggerMesh objects
  delete      Delete components by names
  describe    Show broker status
  dump        Generate Kubernetes manifest
  help        Help about any command
  send-event  Send CloudEvent to the broker
  start       Starts TriggerMesh components
  stop        Stops TriggerMesh components
  version     CLI version information
  watch       Watch events flowing through the broker
```

For the quickstart guide, please visit [docs.triggermesh.io](https://docs.triggermesh.io).

## Contributing

We are happy to review and accept pull requests.

## Commercial Support

TriggerMesh Inc. offers commercial support for the TriggerMesh platform. Email us at <info@triggermesh.com> to get more details.

## License

This software is licensed under the [Apache License, Version 2.0][asl2].

[asl2]: https://www.apache.org/licenses/LICENSE-2.0
