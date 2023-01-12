## tmctl dump

Generate manifest

```
tmctl dump [broker] [flags]
```

### Examples

```
tmctl dump
```

### Options

```
  -h, --help              help for dump
  -o, --output string     Output format (default "yaml")
  -p, --platform string   kubernetes, knative, docker-compose, digitalocean (default "kubernetes")
```

### Options inherited from parent commands

```
      --broker string    Optional broker name.
      --version string   TriggerMesh components version.
```

### SEE ALSO

* [tmctl](tmctl.md)	 - A command line interface to build event-driven applications

