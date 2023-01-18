## tmctl dump

Generate TriggerMesh manifests

```
tmctl dump [broker] -p <kubernetes|knative|docker-compose|digitalocean> [-o json] [flags]
```

### Examples

```
tmctl dump
```

### Options

```
  -i, --do-instance string   DigitalOcean instance size (default "professional-xs")
  -r, --do-region string     DigitalOcean region (default "fra")
  -h, --help                 help for dump
  -o, --output string        Output format (default "yaml")
  -p, --platform string      Target platform. One of kubernetes, knative, docker-compose, digitalocean (default "kubernetes")
```

### Options inherited from parent commands

```
      --broker string    Optional broker name.
      --version string   TriggerMesh components version.
```

### SEE ALSO

* [tmctl](tmctl.md)	 - A command line interface to build event-driven applications

