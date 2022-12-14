[![Release](https://img.shields.io/github/v/release/triggermesh/tmctl?label=release)](https://github.com/triggermesh/tmctl/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/triggermesh/tmctl)](https://goreportcard.com/report/github.com/triggermesh/tmctl)
[![Slack](https://img.shields.io/badge/Slack-Join%20chat-4a154b?style=flat&logo=slack)](https://join.slack.com/t/triggermesh-community/shared_invite/zt-1kngevosm-MY7kqn9h6bT08hWh8PeltA)

# TriggerMesh CLI

`tmctl` is the TriggerMesh CLI (part of the Shaker project) to create, manage and debug event-driven integration apps. This CLI provides a simple user experience in your local environment and supports further deployment to a  Kubernetes cluster.


For the full documentation of TriggerMesh and its tooling, please visit [docs.triggermesh.io](https://docs.triggermesh.io).

## Requirements

The CLI runs TriggerMesh components locally as containers, therefore Docker engine must be running on the machine where `tmctl` is installed.

## Installation

TriggerMesh CLI can be installed from different sources: brew repository, pre-built binary, or compiled from the source.

### Brew

```
brew install triggermesh/cli/tmctl
```

### Manual install

Download and install the latest version for your platform from the [releases page](https://github.com/triggermesh/tmctl/releases). For example,

```bash
curl -LO https://github.com/triggermesh/tmctl/releases/latest/download/tmctl_macOS_amd64.tar.gz && \
tar -zxf tmctl_macOS_amd64.tar.gz && \
chmod +x tmctl && \
sudo mv tmctl /usr/local/bin/tmctl
```

Alternatively, if you're feeling brave, use the following one-liner to automatically download and install the CLI:

```bash
curl -sSfL https://raw.githubusercontent.com/triggermesh/tmctl/HEAD/hack/install.sh | sh
```

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

The CLI commands provide a way to manage TriggerMesh components locally and deploy them on a Kubernetes cluster without the need to write YAML files. All commands support `--help` argument to get the description and usage:

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
